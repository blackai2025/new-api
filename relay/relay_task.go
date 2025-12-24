package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

/*
Task 任务通过平台、Action 区分任务
*/
func RelayTaskSubmit(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	info.InitChannelMeta(c)
	// ensure TaskRelayInfo is initialized to avoid nil dereference when accessing embedded fields
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	path := c.Request.URL.Path
	if strings.Contains(path, "/v1/videos/") && strings.HasSuffix(path, "/remix") {
		info.Action = constant.TaskActionRemix
	}

	// 提取 remix 任务的 video_id
	if info.Action == constant.TaskActionRemix {
		videoID := c.Param("video_id")
		if strings.TrimSpace(videoID) == "" {
			return service.TaskErrorWrapperLocal(fmt.Errorf("video_id is required"), "invalid_request", http.StatusBadRequest)
		}
		info.OriginTaskID = videoID
	}

	platform := constant.TaskPlatform(c.GetString("platform"))

	// 获取原始任务信息
	if info.OriginTaskID != "" {
		originTask, exist, err := model.GetByTaskId(info.UserId, info.OriginTaskID)
		if err != nil {
			taskErr = service.TaskErrorWrapper(err, "get_origin_task_failed", http.StatusInternalServerError)
			return
		}
		if !exist {
			taskErr = service.TaskErrorWrapperLocal(errors.New("task_origin_not_exist"), "task_not_exist", http.StatusBadRequest)
			return
		}
		if info.OriginModelName == "" {
			if originTask.Properties.OriginModelName != "" {
				info.OriginModelName = originTask.Properties.OriginModelName
			} else if originTask.Properties.UpstreamModelName != "" {
				info.OriginModelName = originTask.Properties.UpstreamModelName
			} else {
				var taskData map[string]interface{}
				_ = json.Unmarshal(originTask.Data, &taskData)
				if m, ok := taskData["model"].(string); ok && m != "" {
					info.OriginModelName = m
					platform = originTask.Platform
				}
			}
		}
		if originTask.ChannelId != info.ChannelId {
			channel, err := model.GetChannelById(originTask.ChannelId, true)
			if err != nil {
				taskErr = service.TaskErrorWrapperLocal(err, "channel_not_found", http.StatusBadRequest)
				return
			}
			if channel.Status != common.ChannelStatusEnabled {
				taskErr = service.TaskErrorWrapperLocal(errors.New("the channel of the origin task is disabled"), "task_channel_disable", http.StatusBadRequest)
				return
			}
			key, _, newAPIError := channel.GetNextEnabledKey()
			if newAPIError != nil {
				taskErr = service.TaskErrorWrapper(newAPIError, "channel_no_available_key", newAPIError.StatusCode)
				return
			}
			common.SetContextKey(c, constant.ContextKeyChannelKey, key)
			common.SetContextKey(c, constant.ContextKeyChannelType, channel.Type)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, channel.GetBaseURL())
			common.SetContextKey(c, constant.ContextKeyChannelId, originTask.ChannelId)

			info.ChannelBaseUrl = channel.GetBaseURL()
			info.ChannelId = originTask.ChannelId
			info.ChannelType = channel.Type
			info.ApiKey = key
			platform = originTask.Platform
		}

		// 使用原始任务的参数
		if info.Action == constant.TaskActionRemix {
			var taskData map[string]interface{}
			_ = json.Unmarshal(originTask.Data, &taskData)
			secondsStr, _ := taskData["seconds"].(string)
			seconds, _ := strconv.Atoi(secondsStr)
			if seconds <= 0 {
				seconds = 4
			}
			sizeStr, _ := taskData["size"].(string)
			if info.PriceData.OtherRatios == nil {
				info.PriceData.OtherRatios = map[string]float64{}
			}
			info.PriceData.OtherRatios["seconds"] = float64(seconds)
			info.PriceData.OtherRatios["size"] = 1
			if sizeStr == "1792x1024" || sizeStr == "1024x1792" {
				info.PriceData.OtherRatios["size"] = 1.666667
			}
		}
	}
	if platform == "" {
		platform = GetTaskPlatform(c)
	}

	info.InitChannelMeta(c)
	adaptor := GetTaskAdaptor(platform)
	if adaptor == nil {
		return service.TaskErrorWrapperLocal(fmt.Errorf("invalid api platform: %s", platform), "invalid_api_platform", http.StatusBadRequest)
	}
	adaptor.Init(info)
	// get & validate taskRequest 获取并验证文本请求
	taskErr = adaptor.ValidateRequestAndSetAction(c, info)
	if taskErr != nil {
		return
	}

	// 模型映射处理 - 将用户请求的模型名映射为上游实际模型名
	modelMapping := c.GetString("model_mapping")
	if modelMapping != "" && modelMapping != "{}" {
		modelMap := make(map[string]string)
		if err := json.Unmarshal([]byte(modelMapping), &modelMap); err == nil {
			// 支持链式模型重定向，最终使用链尾的模型
			currentModel := info.OriginModelName
			visitedModels := map[string]bool{currentModel: true}
			for {
				if mappedModel, exists := modelMap[currentModel]; exists && mappedModel != "" {
					// 检测循环
					if visitedModels[mappedModel] {
						if mappedModel != currentModel {
							common.SysError(fmt.Sprintf("model mapping contains cycle: %s -> %s", currentModel, mappedModel))
						}
						break
					}
					visitedModels[mappedModel] = true
					currentModel = mappedModel
					info.IsModelMapped = true
				} else {
					break
				}
			}
			if info.IsModelMapped {
				info.UpstreamModelName = currentModel
				common.SysLog(fmt.Sprintf("[Task] Model mapped: %s -> %s", info.OriginModelName, info.UpstreamModelName))
			}
		}
	}
	// 如果没有映射，UpstreamModelName 使用 OriginModelName
	if info.UpstreamModelName == "" {
		info.UpstreamModelName = info.OriginModelName
	}

	modelName := info.OriginModelName
	if modelName == "" {
		modelName = service.CoverTaskActionToModelName(platform, info.Action)
	}
	modelPrice, success := ratio_setting.GetModelPrice(modelName, true)
	if !success {
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[modelName]
		if !ok {
			modelPrice = 0.1
		} else {
			modelPrice = defaultPrice
		}
	}

	// 预扣
	groupRatio := ratio_setting.GetGroupRatio(info.UsingGroup)
	var ratio float64
	userGroupRatio, hasUserGroupRatio := ratio_setting.GetGroupGroupRatio(info.UserGroup, info.UsingGroup)
	if hasUserGroupRatio {
		ratio = modelPrice * userGroupRatio
	} else {
		ratio = modelPrice * groupRatio
	}
	// FIXME: 临时修补，支持任务仅按次计费
	if !common.StringsContains(constant.TaskPricePatches, modelName) {
		if len(info.PriceData.OtherRatios) > 0 {
			for _, ra := range info.PriceData.OtherRatios {
				if 1.0 != ra {
					ratio *= ra
				}
			}
		}
	}
	println(fmt.Sprintf("model: %s, model_price: %.4f, group: %s, group_ratio: %.4f, final_ratio: %.4f", modelName, modelPrice, info.UsingGroup, groupRatio, ratio))
	userQuota, err := model.GetUserQuota(info.UserId, false)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
		return
	}
	quota := int(ratio * common.QuotaPerUnit)
	if userQuota-quota < 0 {
		taskErr = service.TaskErrorWrapperLocal(errors.New("user quota is not enough"), "quota_not_enough", http.StatusForbidden)
		return
	}

	// build body
	requestBody, err := adaptor.BuildRequestBody(c, info)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "build_request_failed", http.StatusInternalServerError)
		return
	}
	// do request
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
		return
	}
	// handle response
	if resp != nil && resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		taskErr = service.TaskErrorWrapper(fmt.Errorf(string(responseBody)), "fail_to_fetch_task", resp.StatusCode)
		return
	}

	defer func() {
		// release quota
		if info.ConsumeQuota && taskErr == nil {

			err := service.PostConsumeQuota(info, quota, 0, true)
			if err != nil {
				common.SysLog("error consuming token remain quota: " + err.Error())
			}
			if quota != 0 {
				tokenName := c.GetString("token_name")
				//gRatio := groupRatio
				//if hasUserGroupRatio {
				//	gRatio = userGroupRatio
				//}
				logContent := fmt.Sprintf("操作 %s", info.Action)
				// FIXME: 临时修补，支持任务仅按次计费
				if common.StringsContains(constant.TaskPricePatches, modelName) {
					logContent = fmt.Sprintf("%s，按次计费", logContent)
				} else {
					if len(info.PriceData.OtherRatios) > 0 {
						var contents []string
						for key, ra := range info.PriceData.OtherRatios {
							if 1.0 != ra {
								contents = append(contents, fmt.Sprintf("%s: %.2f", key, ra))
							}
						}
						if len(contents) > 0 {
							logContent = fmt.Sprintf("%s, 计算参数：%s", logContent, strings.Join(contents, ", "))
						}
					}
				}
				other := make(map[string]interface{})
				if c != nil && c.Request != nil && c.Request.URL != nil {
					other["request_path"] = c.Request.URL.Path
				}
				other["model_price"] = modelPrice
				other["group_ratio"] = groupRatio
				if hasUserGroupRatio {
					other["user_group_ratio"] = userGroupRatio
				}
				model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
					ChannelId: info.ChannelId,
					ModelName: modelName,
					TokenName: tokenName,
					Quota:     quota,
					Content:   logContent,
					TokenId:   info.TokenId,
					Group:     info.UsingGroup,
					Other:     other,
				})
				model.UpdateUserUsedQuotaAndRequestCount(info.UserId, quota)
				model.UpdateChannelUsedQuota(info.ChannelId, quota)
			}
		}
	}()

	taskID, taskData, taskErr := adaptor.DoResponse(c, resp, info)
	if taskErr != nil {
		return
	}
	info.ConsumeQuota = true
	// insert task
	task := model.InitTask(platform, info)
	task.TaskID = taskID
	task.Quota = quota
	task.Data = taskData
	task.Action = info.Action

	// APIMart：解析提交响应中的初始状态，立即更新任务状态
	if info.ChannelType == constant.ChannelTypeAPIMart {
		var submitResp struct {
			Code int `json:"code"`
			Data []struct {
				Status    string   `json:"status"`
				TaskID    string   `json:"task_id"`
				ImageURLs []string `json:"image_urls,omitempty"`
				VideoURLs []string `json:"video_urls,omitempty"`
			} `json:"data"`
		}

		if err := json.Unmarshal(taskData, &submitResp); err == nil && len(submitResp.Data) > 0 {
			initialStatus := submitResp.Data[0].Status

			// 映射初始状态（与 adaptor 中的 mapStatus 保持一致）
			switch initialStatus {
			case "submitted":
				task.Status = model.TaskStatusSubmitted
				task.Progress = "0%"
			case "processing", "in_progress":
				task.Status = model.TaskStatusInProgress
				task.Progress = "50%"
			case "succeeded", "completed":
				task.Status = model.TaskStatusSuccess
				task.Progress = "100%"
			case "failed", "error":
				task.Status = model.TaskStatusFailure
				task.Progress = "100%"
			}

			// 如果提交响应中直接包含 URL（快速生成），立即设置
			var resultURL string
			if len(submitResp.Data[0].ImageURLs) > 0 {
				resultURL = submitResp.Data[0].ImageURLs[0]
			} else if len(submitResp.Data[0].VideoURLs) > 0 {
				resultURL = submitResp.Data[0].VideoURLs[0]
			}

			if resultURL != "" {
				task.FailReason = resultURL
				task.Status = model.TaskStatusSuccess
				task.Progress = "100%"
			}
		}
	}

	// 兼容旧的 immediate URL 方式
	if immediateURL, exists := c.Get("apimart_immediate_url"); exists {
		if urlStr, ok := immediateURL.(string); ok && urlStr != "" && task.FailReason == "" {
			task.FailReason = urlStr
			task.Status = model.TaskStatusSuccess
			task.Progress = "100%"
		}
	}

	err = task.Insert()
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "insert_task_failed", http.StatusInternalServerError)
		return
	}
	return nil
}

var fetchRespBuilders = map[int]func(c *gin.Context) (respBody []byte, taskResp *dto.TaskError){
	relayconstant.RelayModeSunoFetchByID:     sunoFetchByIDRespBodyBuilder,
	relayconstant.RelayModeSunoFetch:         sunoFetchRespBodyBuilder,
	relayconstant.RelayModeVideoFetchByID:    videoFetchByIDRespBodyBuilder,
	relayconstant.RelayModeImagesGenerations: imageFetchByIDRespBodyBuilder,
}

func RelayTaskFetch(c *gin.Context, relayMode int) (taskResp *dto.TaskError) {
	respBuilder, ok := fetchRespBuilders[relayMode]
	if !ok {
		taskResp = service.TaskErrorWrapperLocal(errors.New("invalid_relay_mode"), "invalid_relay_mode", http.StatusBadRequest)
	}

	respBody, taskErr := respBuilder(c)
	if taskErr != nil {
		return taskErr
	}
	if len(respBody) == 0 {
		respBody = []byte("{\"code\":\"success\",\"data\":null}")
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	_, err := io.Copy(c.Writer, bytes.NewBuffer(respBody))
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "copy_response_body_failed", http.StatusInternalServerError)
		return
	}
	return
}

func sunoFetchRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	userId := c.GetInt("id")
	condition := struct {
		IDs    []any  `json:"ids"`
		Action string `json:"action"`
	}{}
	err := c.BindJSON(&condition)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest)
		return
	}
	var tasks []any
	if len(condition.IDs) > 0 {
		taskModels, err := model.GetByTaskIds(userId, condition.IDs)
		if err != nil {
			taskResp = service.TaskErrorWrapper(err, "get_tasks_failed", http.StatusInternalServerError)
			return
		}
		for _, task := range taskModels {
			tasks = append(tasks, TaskModel2Dto(task))
		}
	} else {
		tasks = make([]any, 0)
	}
	respBody, err = json.Marshal(dto.TaskResponse[[]any]{
		Code: "success",
		Data: tasks,
	})
	return
}

func sunoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("id")
	userId := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}

	respBody, err = json.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: TaskModel2Dto(originTask),
	})
	return
}

func videoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("task_id")
	if taskId == "" {
		taskId = c.GetString("task_id")
	}
	userId := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}

	func() {
		channelModel, err2 := model.GetChannelById(originTask.ChannelId, true)
		if err2 != nil {
			return
		}
		if channelModel.Type != constant.ChannelTypeVertexAi && channelModel.Type != constant.ChannelTypeGemini {
			return
		}
		baseURL := constant.ChannelBaseURLs[channelModel.Type]
		if channelModel.GetBaseURL() != "" {
			baseURL = channelModel.GetBaseURL()
		}
		proxy := channelModel.GetSetting().Proxy
		adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(channelModel.Type)))
		if adaptor == nil {
			return
		}
		resp, err2 := adaptor.FetchTask(baseURL, channelModel.Key, map[string]any{
			"task_id": originTask.TaskID,
			"action":  originTask.Action,
		}, proxy)
		if err2 != nil || resp == nil {
			return
		}
		defer resp.Body.Close()
		body, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return
		}
		ti, err2 := adaptor.ParseTaskResult(body)
		if err2 == nil && ti != nil {
			if ti.Status != "" {
				originTask.Status = model.TaskStatus(ti.Status)
			}
			if ti.Progress != "" {
				originTask.Progress = ti.Progress
			}
			if ti.Url != "" {
				if strings.HasPrefix(ti.Url, "data:") {
				} else {
					originTask.FailReason = ti.Url
				}
			}
			_ = originTask.Update()
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			format := "mp4"
			if respObj, ok := raw["response"].(map[string]any); ok {
				if vids, ok := respObj["videos"].([]any); ok && len(vids) > 0 {
					if v0, ok := vids[0].(map[string]any); ok {
						if mt, ok := v0["mimeType"].(string); ok && mt != "" {
							if strings.Contains(mt, "mp4") {
								format = "mp4"
							} else {
								format = mt
							}
						}
					}
				}
			}
			status := "processing"
			switch originTask.Status {
			case model.TaskStatusSuccess:
				status = "succeeded"
			case model.TaskStatusFailure:
				status = "failed"
			case model.TaskStatusQueued, model.TaskStatusSubmitted:
				status = "queued"
			}
			if !strings.HasPrefix(c.Request.RequestURI, "/v1/videos/") {
				out := map[string]any{
					"error":    nil,
					"format":   format,
					"metadata": nil,
					"status":   status,
					"task_id":  originTask.TaskID,
					"url":      originTask.FailReason,
				}
				respBody, _ = json.Marshal(dto.TaskResponse[any]{
					Code: "success",
					Data: out,
				})
			}
		}
	}()

	if len(respBody) != 0 {
		return
	}

	if strings.HasPrefix(c.Request.RequestURI, "/v1/videos/") {
		adaptor := GetTaskAdaptor(originTask.Platform)
		if adaptor == nil {
			taskResp = service.TaskErrorWrapperLocal(fmt.Errorf("invalid channel id: %d", originTask.ChannelId), "invalid_channel_id", http.StatusBadRequest)
			return
		}
		if converter, ok := adaptor.(channel.OpenAIVideoConverter); ok {
			openAIVideoData, err := converter.ConvertToOpenAIVideo(originTask)
			if err != nil {
				taskResp = service.TaskErrorWrapper(err, "convert_to_openai_video_failed", http.StatusInternalServerError)
				return
			}
			respBody = openAIVideoData
			return
		}
		taskResp = service.TaskErrorWrapperLocal(errors.New(fmt.Sprintf("not_implemented:%s", originTask.Platform)), "not_implemented", http.StatusNotImplemented)
		return
	}
	respBody, err = json.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: TaskModel2Dto(originTask),
	})
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return
}

func TaskModel2Dto(task *model.Task) *dto.TaskDto {
	return &dto.TaskDto{
		TaskID:     task.TaskID,
		Action:     task.Action,
		Status:     string(task.Status),
		FailReason: task.FailReason,
		SubmitTime: task.SubmitTime,
		StartTime:  task.StartTime,
		FinishTime: task.FinishTime,
		Progress:   task.Progress,
		Data:       task.Data,
	}
}

// imageFetchByIDRespBodyBuilder 图片任务查询处理器
func imageFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("task_id")
	if taskId == "" {
		taskId = c.GetString("task_id")
	}
	userId := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusNotFound)
		return
	}

	// 尝试获取适配器
	adaptor := GetTaskAdaptor(originTask.Platform)

	// 如果找不到适配器（platform 为空或不支持），尝试根据渠道类型推断
	if adaptor == nil && originTask.ChannelId > 0 {
		channelModel, err := model.GetChannelById(originTask.ChannelId, true)
		if err == nil && channelModel != nil {
			platform := inferPlatformFromChannelType(channelModel.Type)
			if platform != "" {
				adaptor = GetTaskAdaptor(platform)
			}
		}
	}

	// 如果有适配器且实现了 OpenAIVideoConverter 接口，使用适配器转换
	if adaptor != nil {
		if converter, ok := adaptor.(channel.OpenAIVideoConverter); ok {
			respBody, err = converter.ConvertToOpenAIVideo(originTask)
			if err != nil {
				taskResp = service.TaskErrorWrapper(err, "convert_response_failed", http.StatusInternalServerError)
				return
			}
			return
		}
	}

	// 回退到统一格式（不依赖适配器）
	respBody, err = buildUnifiedTaskResponse(originTask)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return
}

// inferPlatformFromChannelType 根据渠道类型推断平台
func inferPlatformFromChannelType(channelType int) constant.TaskPlatform {
	switch channelType {
	case constant.ChannelTypeAPIMart:
		return constant.TaskPlatformAPIMart
	case constant.ChannelTypeKieAI:
		return constant.TaskPlatformKieAI
	case constant.ChannelTypeSunoAPI:
		return constant.TaskPlatformSuno
	// 其他渠道类型使用字符串值（用于 GetTaskAdaptor 查找）
	case constant.ChannelTypeSora:
		return constant.TaskPlatform(strconv.Itoa(channelType))
	case constant.ChannelTypeKling:
		return constant.TaskPlatform(strconv.Itoa(channelType))
	case constant.ChannelTypeJimeng:
		return constant.TaskPlatform(strconv.Itoa(channelType))
	case constant.ChannelTypeVidu:
		return constant.TaskPlatform(strconv.Itoa(channelType))
	case constant.ChannelTypeDoubaoVideo:
		return constant.TaskPlatform(strconv.Itoa(channelType))
	case constant.ChannelTypeAli:
		return constant.TaskPlatform(strconv.Itoa(channelType))
	case constant.ChannelTypeGemini, constant.ChannelTypeVertexAi:
		return constant.TaskPlatform(strconv.Itoa(channelType))
	default:
		return ""
	}
}

// buildUnifiedTaskResponse 构建统一的任务响应格式
func buildUnifiedTaskResponse(task *model.Task) ([]byte, error) {
	status := mapTaskStatusToUnified(string(task.Status))
	progress := 0
	if task.Progress != "" {
		progressStr := strings.TrimSuffix(task.Progress, "%")
		progress, _ = strconv.Atoi(progressStr)
	}

	response := map[string]interface{}{
		"id":         task.TaskID,
		"object":     "generation.task",
		"model":      task.Properties.OriginModelName,
		"status":     status,
		"progress":   progress,
		"created_at": task.SubmitTime,
		"metadata":   map[string]interface{}{},
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
		response["result"] = map[string]interface{}{
			"type": resultType,
			"data": []map[string]interface{}{
				{"url": task.FailReason},
			},
		}
		response["expires_at"] = task.FinishTime + 24*3600
	}

	// 如果失败
	if status == "failed" {
		response["error"] = map[string]interface{}{
			"code":    "generation_failed",
			"message": task.FailReason,
		}
	}

	return json.Marshal(response)
}

// mapTaskStatusToUnified 将内部状态映射为统一 API 状态
func mapTaskStatusToUnified(status string) string {
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
