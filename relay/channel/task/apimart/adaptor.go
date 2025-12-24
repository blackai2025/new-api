package apimart

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== 统一响应格式（对外暴露） ====================

// GenerationTaskResponse 统一的异步任务响应格式
type GenerationTaskResponse struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	Model       string                 `json:"model"`
	Status      string                 `json:"status"`
	Progress    int                    `json:"progress"`
	CreatedAt   int64                  `json:"created_at"`
	CompletedAt int64                  `json:"completed_at,omitempty"`
	ExpiresAt   int64                  `json:"expires_at,omitempty"`
	Result      *GenerationResult      `json:"result,omitempty"`
	Error       *GenerationError       `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GenerationResult 生成结果
type GenerationResult struct {
	Type string               `json:"type"` // "image" or "video"
	Data []GenerationDataItem `json:"data"`
}

// GenerationDataItem 生成数据项
type GenerationDataItem struct {
	URL           string `json:"url"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Duration      int    `json:"duration,omitempty"`
	Size          string `json:"size,omitempty"`
	Format        string `json:"format,omitempty"`
}

// GenerationError 错误信息
type GenerationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ==================== APIMart 上游响应结构 ====================

// APIMart 提交响应结构（data 是数组）
type APIMartSubmitResponse struct {
	Code      int               `json:"code"`
	Data      []APIMartTaskData `json:"data"`
	ImageURLs []string          `json:"image_urls,omitempty"` // 快速生成时直接返回
	VideoURLs []string          `json:"video_urls,omitempty"` // 快速生成时直接返回
}

type APIMartTaskData struct {
	Status    string   `json:"status"`
	TaskID    string   `json:"task_id"`
	ImageURLs []string `json:"image_urls,omitempty"` // 任务内的URL
	VideoURLs []string `json:"video_urls,omitempty"` // 任务内的URL
}

// APIMart 查询响应结构（/v1/tasks/{task_id} 返回格式）
type APIMartQueryResponse struct {
	Code int              `json:"code"` // 200 表示成功
	Data APIMartQueryData `json:"data"`
}

type APIMartQueryData struct {
	ID            string        `json:"id"`
	Status        string        `json:"status"`   // "completed", "processing", etc.
	Progress      int           `json:"progress"` // 0-100 整数
	ActualTime    int           `json:"actual_time"`
	EstimatedTime int           `json:"estimated_time"`
	Created       int64         `json:"created"`
	Completed     int64         `json:"completed"`
	Result        APIMartResult `json:"result"`
}

type APIMartResult struct {
	Images []APIMartMedia `json:"images,omitempty"`
	Videos []APIMartMedia `json:"videos,omitempty"`
}

type APIMartMedia struct {
	URL       []string `json:"url"`
	ExpiresAt int64    `json:"expires_at,omitempty"`
}

// 统一任务适配器
type TaskAdaptor struct {
	ChannelType int
	taskType    string // "video" or "image"
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	// 自动检测任务类型
	path := c.Request.URL.Path

	if strings.Contains(path, "/videos/") {
		a.taskType = "video"
		return a.validateVideoRequest(c, info)
	} else if strings.Contains(path, "/images/") {
		a.taskType = "image"
		return a.validateImageRequest(c, info)
	}

	return service.TaskErrorWrapperLocal(
		fmt.Errorf("unknown task type from path: %s", path),
		"invalid_request",
		http.StatusBadRequest,
	)
}

func (a *TaskAdaptor) validateVideoRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var videoReq *relaycommon.TaskSubmitReq
	err := common.UnmarshalBodyReusable(c, &videoReq)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	if videoReq.Model == "" || videoReq.Prompt == "" {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("model and prompt are required"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}

	info.Action = constant.TaskActionVideoGenerate
	c.Set("task_request", videoReq)
	return nil
}

func (a *TaskAdaptor) validateImageRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var imageReq *dto.ImageRequest
	err := common.UnmarshalBodyReusable(c, &imageReq)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	if imageReq.Model == "" || imageReq.Prompt == "" {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("model and prompt are required"),
			"invalid_request",
			http.StatusBadRequest,
		)
	}

	info.Action = constant.TaskActionImageGenerate
	c.Set("task_request", imageReq)
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := info.ChannelBaseUrl

	// 根据任务类型构建 URL
	if a.taskType == "video" {
		return fmt.Sprintf("%s/v1/videos/generations", baseURL), nil
	} else if a.taskType == "image" {
		return fmt.Sprintf("%s/v1/images/generations", baseURL), nil
	}

	return "", fmt.Errorf("unknown task type: %s", a.taskType)
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	taskReq, ok := c.Get("task_request")
	if !ok {
		return nil, fmt.Errorf("task_request not found")
	}

	// 使用映射后的模型名
	upstreamModel := info.UpstreamModelName
	if upstreamModel == "" {
		upstreamModel = info.OriginModelName
	}

	// 根据任务类型替换模型名
	if a.taskType == "video" {
		videoReq := taskReq.(*relaycommon.TaskSubmitReq)
		videoReq.Model = upstreamModel
		data, err := json.Marshal(videoReq)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	} else {
		imageReq := taskReq.(*dto.ImageRequest)
		imageReq.Model = upstreamModel
		data, err := json.Marshal(imageReq)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}

	// 解析 APIMart 提交响应
	var apiResponse APIMartSubmitResponse
	err = json.Unmarshal(responseBody, &apiResponse)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}

	if apiResponse.Code != 200 {
		return "", nil, service.TaskErrorWrapper(
			fmt.Errorf("upstream error, code: %d", apiResponse.Code),
			"upstream_error",
			http.StatusInternalServerError,
		)
	}

	// 提取任务ID和可能的立即返回的 URL
	var immediateURL string
	if len(apiResponse.Data) > 0 {
		taskID = apiResponse.Data[0].TaskID

		// 如果提交响应中直接包含 URL（快速生成），保存到 context 供后续使用
		if len(apiResponse.Data[0].ImageURLs) > 0 {
			immediateURL = apiResponse.Data[0].ImageURLs[0]
		} else if len(apiResponse.Data[0].VideoURLs) > 0 {
			immediateURL = apiResponse.Data[0].VideoURLs[0]
		} else if len(apiResponse.ImageURLs) > 0 {
			immediateURL = apiResponse.ImageURLs[0]
		} else if len(apiResponse.VideoURLs) > 0 {
			immediateURL = apiResponse.VideoURLs[0]
		}

		if immediateURL != "" {
			c.Set("apimart_immediate_url", immediateURL)
		}
	}

	// 获取模型名称
	modelName := info.OriginModelName

	// 构建统一响应格式
	unifiedResponse := GenerationTaskResponse{
		ID:        taskID,
		Object:    "generation.task",
		Model:     modelName,
		Status:    mapStatusToUnified(apiResponse.Data[0].Status),
		Progress:  0,
		CreatedAt: time.Now().Unix(),
		Metadata:  make(map[string]interface{}),
	}

	// 如果有立即返回的 URL，标记为已完成
	if immediateURL != "" {
		unifiedResponse.Status = "completed"
		unifiedResponse.Progress = 100
		unifiedResponse.CompletedAt = time.Now().Unix()
		unifiedResponse.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
		unifiedResponse.Result = &GenerationResult{
			Type: a.taskType,
			Data: []GenerationDataItem{
				{URL: immediateURL},
			},
		}
	}

	// 序列化统一响应
	unifiedResponseData, err := json.Marshal(unifiedResponse)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}

	taskData = unifiedResponseData

	// 返回统一格式给用户
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(unifiedResponseData)

	return taskID, taskData, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id not found")
	}

	// 构建查询 URL（apiMart 使用统一的任务查询端点 /v1/tasks/{task_id}）
	requestUrl := fmt.Sprintf("%s/v1/tasks/%s", baseUrl, taskID)

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}

	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var apiResponse APIMartQueryResponse
	err := json.Unmarshal(respBody, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	// 检查是否成功
	if apiResponse.Code != 200 {
		return nil, fmt.Errorf("upstream error, code: %d", apiResponse.Code)
	}

	// 统一状态映射
	status := mapStatus(apiResponse.Data.Status)
	progress := fmt.Sprintf("%d%%", apiResponse.Data.Progress)

	// 提取 URL：优先图片，其次视频
	var resultURL string
	if len(apiResponse.Data.Result.Images) > 0 && len(apiResponse.Data.Result.Images[0].URL) > 0 {
		resultURL = apiResponse.Data.Result.Images[0].URL[0]
	} else if len(apiResponse.Data.Result.Videos) > 0 && len(apiResponse.Data.Result.Videos[0].URL) > 0 {
		resultURL = apiResponse.Data.Result.Videos[0].URL[0]
	}

	return &relaycommon.TaskInfo{
		Status:   status,
		Progress: progress,
		Url:      resultURL,
	}, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return append(VideoModelList, ImageModelList...)
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// ConvertToOpenAIVideo 实现 OpenAIVideoConverter 接口，将任务转换为统一响应格式
func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	status := mapStatusToUnified(string(originTask.Status))

	response := &GenerationTaskResponse{
		ID:        originTask.TaskID,
		Object:    "generation.task",
		Model:     originTask.Properties.OriginModelName,
		Status:    status,
		CreatedAt: originTask.SubmitTime,
		Metadata:  make(map[string]interface{}),
	}

	// 解析进度
	if originTask.Progress != "" {
		progressStr := strings.TrimSuffix(originTask.Progress, "%")
		if p, err := strconv.Atoi(progressStr); err == nil {
			response.Progress = p
		}
	}

	// 如果完成，填充结果
	if status == "completed" {
		response.CompletedAt = originTask.FinishTime
		response.Progress = 100
		response.ExpiresAt = originTask.FinishTime + 24*3600 // 24小时后过期

		// FailReason 存储的是结果 URL
		if originTask.FailReason != "" && !strings.HasPrefix(originTask.FailReason, "error") {
			// 根据 action 判断类型
			resultType := "video"
			if originTask.Action == constant.TaskActionImageGenerate {
				resultType = "image"
			}

			response.Result = &GenerationResult{
				Type: resultType,
				Data: []GenerationDataItem{
					{URL: originTask.FailReason},
				},
			}

			if resultType == "video" {
				response.Result.Data[0].Format = "mp4"
			}
		}
	}

	// 如果失败，填充错误信息
	if status == "failed" {
		response.Error = &GenerationError{
			Code:    "generation_failed",
			Message: originTask.FailReason,
		}
		if response.Error.Message == "" {
			response.Error.Message = "Task failed during generation"
		}
	}

	return json.Marshal(response)
}

// ConvertQueryToUnified 将查询响应转换为统一格式
func (a *TaskAdaptor) ConvertQueryToUnified(apiResponse *APIMartQueryResponse, modelName string) *GenerationTaskResponse {
	data := apiResponse.Data
	status := mapStatusToUnified(data.Status)

	response := &GenerationTaskResponse{
		ID:        data.ID,
		Object:    "generation.task",
		Model:     modelName,
		Status:    status,
		Progress:  data.Progress,
		CreatedAt: data.Created,
		Metadata:  make(map[string]interface{}),
	}

	// 如果完成，填充结果
	if status == "completed" {
		response.CompletedAt = data.Completed
		response.Progress = 100

		// 检测结果类型并构建 result
		if len(data.Result.Images) > 0 && len(data.Result.Images[0].URL) > 0 {
			response.Result = &GenerationResult{
				Type: "image",
				Data: make([]GenerationDataItem, 0),
			}
			for _, img := range data.Result.Images {
				for _, url := range img.URL {
					response.Result.Data = append(response.Result.Data, GenerationDataItem{
						URL: url,
					})
				}
				if img.ExpiresAt > 0 {
					response.ExpiresAt = img.ExpiresAt
				}
			}
		} else if len(data.Result.Videos) > 0 && len(data.Result.Videos[0].URL) > 0 {
			response.Result = &GenerationResult{
				Type: "video",
				Data: make([]GenerationDataItem, 0),
			}
			for _, vid := range data.Result.Videos {
				for _, url := range vid.URL {
					response.Result.Data = append(response.Result.Data, GenerationDataItem{
						URL:    url,
						Format: "mp4",
					})
				}
				if vid.ExpiresAt > 0 {
					response.ExpiresAt = vid.ExpiresAt
				}
			}
		}

		// 如果没有设置过期时间，默认 24 小时
		if response.ExpiresAt == 0 {
			response.ExpiresAt = time.Now().Add(24 * time.Hour).Unix()
		}
	}

	// 如果失败，填充错误信息
	if status == "failed" {
		response.Error = &GenerationError{
			Code:    "generation_failed",
			Message: "Task failed during generation",
		}
	}

	return response
}

// mapStatus 状态映射（用于内部任务管理）
func mapStatus(status string) string {
	switch strings.ToLower(status) {
	case "submitted", "queued", "pending":
		return "SUBMITTED"
	case "processing", "in_progress", "running":
		return "IN_PROGRESS"
	case "succeeded", "completed", "success":
		return "SUCCESS"
	case "failed", "error", "failure":
		return "FAILURE"
	default:
		return "UNKNOWN"
	}
}

// mapStatusToUnified 状态映射为统一 API 格式（对外暴露）
func mapStatusToUnified(status string) string {
	switch strings.ToLower(status) {
	case "submitted", "queued", "pending":
		return "queued"
	case "processing", "in_progress", "running":
		return "in_progress"
	case "succeeded", "completed", "success":
		return "completed"
	case "failed", "error", "failure":
		return "failed"
	default:
		return "queued"
	}
}

func calculateProgress(status string) string {
	switch status {
	case "SUCCESS", "completed":
		return "100%"
	case "IN_PROGRESS", "in_progress":
		return "50%"
	default:
		return "0%"
	}
}

