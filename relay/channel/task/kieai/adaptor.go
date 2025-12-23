package kieai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// ============================
// Request structures
// ============================

// KieCreateTaskRequest Kie.ai 通用任务 API 请求
type KieCreateTaskRequest struct {
	Model       string         `json:"model"`
	Input       map[string]any `json:"input"`
	CallBackUrl string         `json:"callBackUrl,omitempty"`
}

// Veo3GenerateRequest Veo3 专用 API 请求
type Veo3GenerateRequest struct {
	Prompt            string   `json:"prompt"`
	Model             string   `json:"model"`
	ImageUrls         []string `json:"imageUrls,omitempty"`
	AspectRatio       string   `json:"aspectRatio,omitempty"`
	GenerationType    string   `json:"generationType,omitempty"`
	Seeds             int      `json:"seeds,omitempty"`
	EnableTranslation bool     `json:"enableTranslation,omitempty"`
	CallBackUrl       string   `json:"callBackUrl,omitempty"`
}

// ============================
// Response structures
// ============================

// KieCreateTaskResponse Kie.ai 创建任务响应
type KieCreateTaskResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskId string `json:"taskId"`
	} `json:"data"`
}

// KieQueryTaskResponse Kie.ai 查询任务响应
type KieQueryTaskResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskId       string `json:"taskId"`
		Model        string `json:"model"`
		State        string `json:"state"` // waiting, success, fail
		Param        string `json:"param"`
		ResultJson   string `json:"resultJson"`
		FailCode     string `json:"failCode"`
		FailMsg      string `json:"failMsg"`
		CostTime     int64  `json:"costTime"`
		CompleteTime int64  `json:"completeTime"`
		CreateTime   int64  `json:"createTime"`
	} `json:"data"`
}

// KieResultJson 结果 JSON 结构
type KieResultJson struct {
	ResultUrls   []string `json:"resultUrls,omitempty"`
	ResultObject any      `json:"resultObject,omitempty"`
}

// Veo3QueryTaskResponse Veo3 专用查询响应
type Veo3QueryTaskResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskId       string `json:"taskId"`
		ParamJson    string `json:"paramJson"`
		CompleteTime string `json:"completeTime"`
		Response     struct {
			TaskId     string   `json:"taskId"`
			ResultUrls []string `json:"resultUrls"`
			OriginUrls []string `json:"originUrls"`
			Resolution string   `json:"resolution"`
		} `json:"response"`
		SuccessFlag  int    `json:"successFlag"` // 0=生成中, 1=成功, 2=失败, 3=生成失败
		ErrorCode    *int   `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		CreateTime   string `json:"createTime"`
		FallbackFlag bool   `json:"fallbackFlag"`
	} `json:"data"`
}

// ============================
// TaskAdaptor implementation
// ============================

type TaskAdaptor struct {
	ChannelType int
	taskType    string // "video" or "image"
	modelName   string // 保存模型名称用于路由
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
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

	// 保存模型名称用于后续路由
	a.modelName = videoReq.Model

	// 对于 image-to-video 模型，检查是否有图片
	if IsSoraImageToVideoModel(videoReq.Model) {
		if videoReq.Image == "" && len(videoReq.Images) == 0 {
			return service.TaskErrorWrapperLocal(
				fmt.Errorf("image is required for image-to-video model"),
				"invalid_request",
				http.StatusBadRequest,
			)
		}
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

	a.modelName = imageReq.Model
	info.Action = constant.TaskActionImageGenerate
	c.Set("task_request", imageReq)
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	baseURL := info.ChannelBaseUrl

	// 根据模型类型选择 API 端点
	if IsVeo3Model(a.modelName) {
		return fmt.Sprintf("%s/api/v1/veo/generate", baseURL), nil
	}

	// Sora 和图片模型使用通用任务 API
	return fmt.Sprintf("%s/api/v1/jobs/createTask", baseURL), nil
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

	var reqBody []byte
	var err error

	if a.taskType == "video" {
		videoReq := taskReq.(*relaycommon.TaskSubmitReq)

		if IsVeo3Model(videoReq.Model) {
			// Veo3 使用专用 API 格式
			veo3Req := a.buildVeo3Request(videoReq)
			reqBody, err = json.Marshal(veo3Req)
		} else {
			// Sora 使用通用任务 API 格式
			kieReq := a.buildSoraRequest(videoReq)
			reqBody, err = json.Marshal(kieReq)
		}
	} else {
		imageReq := taskReq.(*dto.ImageRequest)
		kieReq := a.buildImageRequest(imageReq)
		reqBody, err = json.Marshal(kieReq)
	}

	if err != nil {
		return nil, err
	}
	return bytes.NewReader(reqBody), nil
}

// buildSoraRequest 构建 Sora 系列模型的请求
func (a *TaskAdaptor) buildSoraRequest(req *relaycommon.TaskSubmitReq) KieCreateTaskRequest {
	input := map[string]any{
		"prompt": req.Prompt,
	}

	// 处理时长参数
	if req.Duration > 0 {
		if req.Duration >= 15 {
			input["n_frames"] = "15"
		} else {
			input["n_frames"] = "10"
		}
	}

	// 处理宽高比
	if req.Size != "" {
		aspectRatio := "landscape"
		if req.Size == "9:16" || req.Size == "portrait" {
			aspectRatio = "portrait"
		}
		input["aspect_ratio"] = aspectRatio
	}

	// 对于 image-to-video 模型，添加图片参数
	if IsSoraImageToVideoModel(req.Model) {
		if req.Image != "" {
			input["image_input"] = []string{req.Image}
		} else if len(req.Images) > 0 {
			input["image_input"] = req.Images
		}
	}

	// 默认去水印
	input["remove_watermark"] = true

	return KieCreateTaskRequest{
		Model: req.Model,
		Input: input,
	}
}

// buildVeo3Request 构建 Veo3 系列模型的请求
func (a *TaskAdaptor) buildVeo3Request(req *relaycommon.TaskSubmitReq) Veo3GenerateRequest {
	veo3Req := Veo3GenerateRequest{
		Prompt:            req.Prompt,
		Model:             req.Model,
		EnableTranslation: true, // 默认启用翻译
	}

	// 处理宽高比
	if req.Size != "" {
		if req.Size == "9:16" || req.Size == "portrait" {
			veo3Req.AspectRatio = "9:16"
		} else {
			veo3Req.AspectRatio = "16:9"
		}
	} else {
		veo3Req.AspectRatio = "16:9" // 默认横屏
	}

	// 处理图片（如果有）
	if req.Image != "" {
		veo3Req.ImageUrls = []string{req.Image}
		veo3Req.GenerationType = "FIRST_AND_LAST_FRAMES_2_VIDEO"
	} else if len(req.Images) > 0 {
		veo3Req.ImageUrls = req.Images
		if len(req.Images) >= 2 {
			veo3Req.GenerationType = "FIRST_AND_LAST_FRAMES_2_VIDEO"
		} else {
			veo3Req.GenerationType = "FIRST_AND_LAST_FRAMES_2_VIDEO"
		}
	} else {
		veo3Req.GenerationType = "TEXT_2_VIDEO"
	}

	return veo3Req
}

// buildImageRequest 构建图片生成请求
func (a *TaskAdaptor) buildImageRequest(req *dto.ImageRequest) KieCreateTaskRequest {
	input := map[string]any{
		"prompt": req.Prompt,
	}

	// 处理尺寸/宽高比
	if req.Size != "" {
		input["aspect_ratio"] = req.Size
	}

	// 处理质量参数
	if req.Quality == "hd" || req.Quality == "high" {
		input["resolution"] = "2K"
	} else {
		input["resolution"] = "1K"
	}

	// 输出格式
	input["output_format"] = "png"

	return KieCreateTaskRequest{
		Model: req.Model,
		Input: input,
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

	var kieResp KieCreateTaskResponse
	err = json.Unmarshal(responseBody, &kieResp)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}

	if kieResp.Code != 200 {
		return "", nil, service.TaskErrorWrapper(
			fmt.Errorf("upstream error: %s (code: %d)", kieResp.Msg, kieResp.Code),
			"upstream_error",
			http.StatusInternalServerError,
		)
	}

	taskID = kieResp.Data.TaskId
	taskData = responseBody

	// 构建统一格式的响应返回给用户
	unifiedResp := map[string]any{
		"id":         taskID,
		"object":     "generation.task",
		"model":      info.OriginModelName,
		"status":     "queued",
		"progress":   0,
		"created_at": time.Now().Unix(),
		"metadata":   map[string]any{},
	}

	unifiedData, _ := json.Marshal(unifiedResp)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(unifiedData)

	return taskID, taskData, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("task_id not found")
	}

	// 获取模型名称，用于选择正确的查询端点
	modelName, _ := body["model"].(string)

	// 根据模型类型选择查询端点
	var requestUrl string
	if IsVeo3Model(modelName) {
		// Veo3 使用专用查询端点
		requestUrl = fmt.Sprintf("%s/api/v1/veo/record-info?taskId=%s", baseUrl, taskID)
	} else {
		// Sora 和图片模型使用通用查询端点
		requestUrl = fmt.Sprintf("%s/api/v1/jobs/recordInfo?taskId=%s", baseUrl, taskID)
	}

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
	// 先尝试解析为通用格式
	var rawResp map[string]any
	if err := json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	// 检查 code
	code, _ := rawResp["code"].(float64)
	if int(code) != 200 {
		msg, _ := rawResp["msg"].(string)
		return nil, fmt.Errorf("upstream error: %s (code: %d)", msg, int(code))
	}

	// 检查 data 中是否有 successFlag 字段（Veo3 格式）
	data, _ := rawResp["data"].(map[string]any)
	if _, hasSuccessFlag := data["successFlag"]; hasSuccessFlag {
		return a.parseVeo3TaskResult(respBody)
	}

	// 否则按通用格式解析
	return a.parseGenericTaskResult(respBody)
}

// parseGenericTaskResult 解析通用 API 响应
func (a *TaskAdaptor) parseGenericTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var kieResp KieQueryTaskResponse
	err := json.Unmarshal(respBody, &kieResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	// 状态映射
	status := mapStatus(kieResp.Data.State)
	progress := "0%"
	switch status {
	case "SUCCESS":
		progress = "100%"
	case "IN_PROGRESS":
		progress = "50%"
	}

	// 解析结果 URL
	var resultURL string
	if kieResp.Data.ResultJson != "" {
		var result KieResultJson
		if err := json.Unmarshal([]byte(kieResp.Data.ResultJson), &result); err == nil {
			if len(result.ResultUrls) > 0 {
				resultURL = result.ResultUrls[0]
			}
		}
	}

	return &relaycommon.TaskInfo{
		Status:   status,
		Progress: progress,
		Url:      resultURL,
	}, nil
}

// parseVeo3TaskResult 解析 Veo3 专用 API 响应
func (a *TaskAdaptor) parseVeo3TaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var veoResp Veo3QueryTaskResponse
	err := json.Unmarshal(respBody, &veoResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal veo3 response failed: %w", err)
	}

	// successFlag 状态映射: 0=生成中, 1=成功, 2=失败, 3=生成失败
	var status string
	var progress string
	switch veoResp.Data.SuccessFlag {
	case 0:
		status = "IN_PROGRESS"
		progress = "50%"
	case 1:
		status = "SUCCESS"
		progress = "100%"
	case 2, 3:
		status = "FAILURE"
		progress = "100%"
	default:
		status = "SUBMITTED"
		progress = "0%"
	}

	// 提取结果 URL
	var resultURL string
	if len(veoResp.Data.Response.ResultUrls) > 0 {
		resultURL = veoResp.Data.Response.ResultUrls[0]
	}

	return &relaycommon.TaskInfo{
		Status:   status,
		Progress: progress,
		Url:      resultURL,
	}, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return GetAllModels()
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

// ConvertToOpenAIVideo 实现 OpenAIVideoConverter 接口
func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	status := mapTaskStatusToAPI(string(task.Status))
	progress := 0
	if task.Progress != "" {
		progressStr := strings.TrimSuffix(task.Progress, "%")
		fmt.Sscanf(progressStr, "%d", &progress)
	}

	response := map[string]any{
		"id":         task.TaskID,
		"object":     "generation.task",
		"model":      task.Properties.OriginModelName,
		"status":     status,
		"progress":   progress,
		"created_at": task.SubmitTime,
		"metadata":   map[string]any{},
	}

	if task.FinishTime > 0 {
		response["completed_at"] = task.FinishTime
	}

	// 如果成功且有结果 URL
	if status == "completed" && task.FailReason != "" && !strings.HasPrefix(task.FailReason, "error") {
		resultType := "image"
		if task.Action == constant.TaskActionVideoGenerate {
			resultType = "video"
		}
		response["result"] = map[string]any{
			"type": resultType,
			"data": []map[string]any{
				{"url": task.FailReason},
			},
		}
		response["expires_at"] = task.FinishTime + 24*3600
	}

	// 如果失败
	if status == "failed" {
		response["error"] = map[string]any{
			"code":    "generation_failed",
			"message": task.FailReason,
		}
	}

	return json.Marshal(response)
}

// mapStatus 将 Kie.ai 状态映射为内部状态
func mapStatus(state string) string {
	switch strings.ToLower(state) {
	case "waiting":
		return "SUBMITTED"
	case "success":
		return "SUCCESS"
	case "fail":
		return "FAILURE"
	default:
		return "UNKNOWN"
	}
}

// mapTaskStatusToAPI 将内部状态映射为 API 状态
func mapTaskStatusToAPI(status string) string {
	switch status {
	case string(model.TaskStatusNotStart), string(model.TaskStatusQueued), string(model.TaskStatusSubmitted):
		return "queued"
	case string(model.TaskStatusInProgress):
		return "in_progress"
	case string(model.TaskStatusSuccess):
		return "completed"
	case string(model.TaskStatusFailure):
		return "failed"
	default:
		return "queued"
	}
}
