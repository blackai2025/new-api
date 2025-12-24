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

// Gpt4oImageRequest GPT-4o Image 专用 API 请求
type Gpt4oImageRequest struct {
	Prompt         string   `json:"prompt,omitempty"`
	FilesUrl       []string `json:"filesUrl,omitempty"`
	Size           string   `json:"size"`
	NVariants      int      `json:"nVariants,omitempty"`
	MaskUrl        string   `json:"maskUrl,omitempty"`
	IsEnhance      bool     `json:"isEnhance,omitempty"`
	UploadCn       bool     `json:"uploadCn,omitempty"`
	EnableFallback bool     `json:"enableFallback,omitempty"`
	CallBackUrl    string   `json:"callBackUrl,omitempty"`
}

// Gpt4oImageQueryResponse GPT-4o Image 专用查询响应
type Gpt4oImageQueryResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TaskId       string `json:"taskId"`
		ParamJson    string `json:"paramJson"`
		CompleteTime int64  `json:"completeTime"`
		Response     struct {
			ResultUrls []string `json:"resultUrls"`
		} `json:"response"`
		SuccessFlag  int    `json:"successFlag"`
		Status       string `json:"status"` // GENERATING, SUCCESS, CREATE_TASK_FAILED, GENERATE_FAILED
		ErrorCode    *int   `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		CreateTime   int64  `json:"createTime"`
		Progress     string `json:"progress"` // 0.00 - 1.00
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

	// 使用映射后的模型名来判断模型类型
	modelName := info.UpstreamModelName
	if modelName == "" {
		modelName = a.modelName
	}

	// 根据模型类型选择 API 端点
	if IsVeo3Model(modelName) {
		return fmt.Sprintf("%s/api/v1/veo/generate", baseURL), nil
	}

	if IsGpt4oImageModel(modelName) {
		return fmt.Sprintf("%s/api/v1/gpt4o-image/generate", baseURL), nil
	}

	// Sora 和 NanoBanana 图片模型使用通用任务 API
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

	// 使用映射后的模型名
	upstreamModel := info.UpstreamModelName
	if upstreamModel == "" {
		upstreamModel = a.modelName
	}

	var reqBody []byte
	var err error

	if a.taskType == "video" {
		videoReq := taskReq.(*relaycommon.TaskSubmitReq)

		// 使用映射后的模型名来判断和构建请求
		if IsVeo3Model(upstreamModel) {
			// Veo3 使用专用 API 格式
			veo3Req := a.buildVeo3RequestWithModel(videoReq, upstreamModel)
			reqBody, err = json.Marshal(veo3Req)
		} else {
			// Sora 使用通用任务 API 格式
			kieReq := a.buildSoraRequestWithModel(videoReq, upstreamModel)
			reqBody, err = json.Marshal(kieReq)
		}
	} else {
		imageReq := taskReq.(*dto.ImageRequest)

		// GPT-4o Image 使用专用 API 格式
		if IsGpt4oImageModel(upstreamModel) {
			gpt4oReq := a.buildGpt4oImageRequest(imageReq)
			reqBody, err = json.Marshal(gpt4oReq)
		} else {
			// NanoBanana 使用通用任务 API 格式
			kieReq := a.buildImageRequestWithModel(imageReq, upstreamModel)
			reqBody, err = json.Marshal(kieReq)
		}
	}

	if err != nil {
		return nil, err
	}
	return bytes.NewReader(reqBody), nil
}

// buildSoraRequestWithModel 构建 Sora 系列模型的请求（使用映射后的模型名）
func (a *TaskAdaptor) buildSoraRequestWithModel(req *relaycommon.TaskSubmitReq, upstreamModel string) KieCreateTaskRequest {
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
	if IsSoraImageToVideoModel(upstreamModel) {
		if req.Image != "" {
			input["image_input"] = []string{req.Image}
		} else if len(req.Images) > 0 {
			input["image_input"] = req.Images
		}
	}

	// 默认去水印
	input["remove_watermark"] = true

	return KieCreateTaskRequest{
		Model: upstreamModel, // 使用映射后的模型名
		Input: input,
	}
}

// buildVeo3RequestWithModel 构建 Veo3 系列模型的请求（使用映射后的模型名）
func (a *TaskAdaptor) buildVeo3RequestWithModel(req *relaycommon.TaskSubmitReq, upstreamModel string) Veo3GenerateRequest {
	veo3Req := Veo3GenerateRequest{
		Prompt:            req.Prompt,
		Model:             upstreamModel, // 使用映射后的模型名
		EnableTranslation: true,          // 默认启用翻译
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

// buildImageRequestWithModel 构建图片生成请求（使用映射后的模型名）
func (a *TaskAdaptor) buildImageRequestWithModel(req *dto.ImageRequest, upstreamModel string) KieCreateTaskRequest {
	input := map[string]any{
		"prompt": req.Prompt,
	}

	// 根据模型类型设置不同的参数
	if upstreamModel == "google/nano-banana" {
		// google/nano-banana: 使用 image_size 参数
		if req.Size != "" {
			input["image_size"] = mapSizeToImageSize(req.Size)
		} else {
			input["image_size"] = "1:1"
		}
		input["output_format"] = "png"

		return KieCreateTaskRequest{
			Model: "google/nano-banana",
			Input: input,
		}
	}

	// seedream/4.5-text-to-image: 使用 aspect_ratio 和 quality (basic/high) 参数
	if IsSeedream45Model(upstreamModel) {
		if req.Size != "" {
			input["aspect_ratio"] = mapSizeToImageSize(req.Size)
		} else {
			input["aspect_ratio"] = "1:1"
		}

		// 处理 quality 参数 (basic=2K, high=4K)
		if req.Quality == "hd" || req.Quality == "high" || req.Quality == "4k" {
			input["quality"] = "high"
		} else {
			input["quality"] = "basic"
		}

		return KieCreateTaskRequest{
			Model: upstreamModel,
			Input: input,
		}
	}

	// bytedance/seedream-v4-text-to-image: 使用特殊的 image_size 和 image_resolution 参数
	if IsSeedreamModel(upstreamModel) {
		if req.Size != "" {
			input["image_size"] = mapSizeToSeedreamSize(req.Size)
		} else {
			input["image_size"] = "square_hd"
		}

		// 处理分辨率参数
		if req.Quality == "hd" || req.Quality == "high" {
			input["image_resolution"] = "2K"
		} else if req.Quality == "4k" {
			input["image_resolution"] = "4K"
		} else {
			input["image_resolution"] = "1K"
		}

		// 处理生成数量（1-6）
		if req.N > 0 && req.N <= 6 {
			input["max_images"] = int(req.N)
		}

		return KieCreateTaskRequest{
			Model: upstreamModel,
			Input: input,
		}
	}

	// nano-banana-pro: 使用 aspect_ratio + resolution 参数
	if req.Size != "" {
		input["aspect_ratio"] = mapSizeToImageSize(req.Size)
	} else {
		input["aspect_ratio"] = "1:1"
	}

	// 处理质量/分辨率参数
	if req.Quality == "hd" || req.Quality == "high" {
		input["resolution"] = "2K"
	} else {
		input["resolution"] = "1K"
	}

	// 输出格式
	input["output_format"] = "png"

	return KieCreateTaskRequest{
		Model: upstreamModel,
		Input: input,
	}
}

// mapSizeToSeedreamSize 将 OpenAI 尺寸映射为 Seedream 的 image_size 格式
func mapSizeToSeedreamSize(size string) string {
	switch size {
	case "1024x1024", "1:1", "square":
		return "square"
	case "square_hd":
		return "square_hd"
	case "1792x1024", "16:9", "landscape":
		return "landscape_16_9"
	case "1024x1792", "9:16", "portrait":
		return "portrait_16_9"
	case "4:3", "landscape_4_3":
		return "landscape_4_3"
	case "3:4", "portrait_4_3":
		return "portrait_4_3"
	case "3:2", "landscape_3_2":
		return "landscape_3_2"
	case "2:3", "portrait_3_2":
		return "portrait_3_2"
	case "21:9", "landscape_21_9":
		return "landscape_21_9"
	default:
		return "square_hd"
	}
}

// mapSizeToImageSize 将 OpenAI 尺寸映射为 Kie.ai 比例
func mapSizeToImageSize(size string) string {
	switch size {
	case "1024x1024", "1:1", "square":
		return "1:1"
	case "1792x1024", "16:9", "landscape":
		return "16:9"
	case "1024x1792", "9:16", "portrait":
		return "9:16"
	case "3:2":
		return "3:2"
	case "2:3":
		return "2:3"
	case "3:4":
		return "3:4"
	case "4:3":
		return "4:3"
	case "4:5":
		return "4:5"
	case "5:4":
		return "5:4"
	case "21:9":
		return "21:9"
	default:
		return "1:1"
	}
}

// buildGpt4oImageRequest 构建 GPT-4o Image 请求
func (a *TaskAdaptor) buildGpt4oImageRequest(req *dto.ImageRequest) Gpt4oImageRequest {
	gpt4oReq := Gpt4oImageRequest{
		Prompt: req.Prompt,
		Size:   mapSizeToGpt4oRatio(req.Size),
	}

	// 处理生成数量（支持 1, 2, 4）
	if req.N > 0 {
		switch req.N {
		case 1, 2, 4:
			gpt4oReq.NVariants = int(req.N)
		default:
			// 向下取整到最近的有效值
			if req.N >= 4 {
				gpt4oReq.NVariants = 4
			} else if req.N >= 2 {
				gpt4oReq.NVariants = 2
			} else {
				gpt4oReq.NVariants = 1
			}
		}
	}

	return gpt4oReq
}

// mapSizeToGpt4oRatio 将 OpenAI 标准尺寸映射为 GPT-4o Image 比例
func mapSizeToGpt4oRatio(size string) string {
	switch size {
	case "1024x1024", "1:1", "square":
		return "1:1"
	case "1792x1024", "3:2", "landscape":
		return "3:2"
	case "1024x1792", "2:3", "portrait":
		return "2:3"
	default:
		return "1:1" // 默认正方形
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
	} else if IsGpt4oImageModel(modelName) {
		// GPT-4o Image 使用专用查询端点
		requestUrl = fmt.Sprintf("%s/api/v1/gpt4o-image/record-info?taskId=%s", baseUrl, taskID)
	} else {
		// Sora 和 NanoBanana 图片模型使用通用查询端点
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

	// 检查 data 中的字段来区分响应类型
	data, _ := rawResp["data"].(map[string]any)

	// GPT-4o Image 格式：有 status 字段（字符串类型）
	if status, hasStatus := data["status"].(string); hasStatus && status != "" {
		return a.parseGpt4oImageTaskResult(respBody)
	}

	// Veo3 格式：有 successFlag 字段
	if _, hasSuccessFlag := data["successFlag"]; hasSuccessFlag {
		return a.parseVeo3TaskResult(respBody)
	}

	// 否则按通用格式解析（Sora/NanoBanana）
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

// parseGpt4oImageTaskResult 解析 GPT-4o Image 专用 API 响应
func (a *TaskAdaptor) parseGpt4oImageTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var gpt4oResp Gpt4oImageQueryResponse
	err := json.Unmarshal(respBody, &gpt4oResp)
	if err != nil {
		return nil, fmt.Errorf("unmarshal gpt4o image response failed: %w", err)
	}

	// 状态映射: GENERATING, SUCCESS, CREATE_TASK_FAILED, GENERATE_FAILED
	var status string
	var progress string
	switch gpt4oResp.Data.Status {
	case "GENERATING":
		status = "IN_PROGRESS"
		// 解析进度（0.00 - 1.00 转换为百分比）
		if gpt4oResp.Data.Progress != "" {
			var p float64
			if _, err := fmt.Sscanf(gpt4oResp.Data.Progress, "%f", &p); err == nil {
				progress = fmt.Sprintf("%d%%", int(p*100))
			} else {
				progress = "50%"
			}
		} else {
			progress = "50%"
		}
	case "SUCCESS":
		status = "SUCCESS"
		progress = "100%"
	case "CREATE_TASK_FAILED", "GENERATE_FAILED":
		status = "FAILURE"
		progress = "100%"
	default:
		status = "SUBMITTED"
		progress = "0%"
	}

	// 提取结果 URL
	var resultURL string
	if len(gpt4oResp.Data.Response.ResultUrls) > 0 {
		resultURL = gpt4oResp.Data.Response.ResultUrls[0]
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
