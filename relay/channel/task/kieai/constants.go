package kieai

const ChannelName = "KieAI"

// ============================
// Sora 系列模型（通用 API: /api/v1/jobs/createTask）
// ============================

// SoraTextToVideoModels 文生视频模型
var SoraTextToVideoModels = []string{
	"sora-2-text-to-video",
	"sora-2-pro-text-to-video",
}

// SoraImageToVideoModels 图生视频模型
var SoraImageToVideoModels = []string{
	"sora-2-image-to-video",
	"sora-2-pro-image-to-video",
}

// ============================
// Veo3 系列模型（专用 API: /api/v1/veo/generate）
// ============================

var Veo3ModelList = []string{
	"veo3",
	"veo3_fast",
}

// ============================
// 图片生成模型（通用 API: /api/v1/jobs/createTask）
// ============================

var ImageModelList = []string{
	"nano-banana-pro",
	"nano-banana",
}

// ============================
// GPT-4o Image 模型（专用 API: /api/v1/gpt4o-image/generate）
// ============================

var Gpt4oImageModelList = []string{
	"gpt-4o-image",
}

// ============================
// 判断函数
// ============================

// IsSoraModel 判断是否为 Sora 系列模型
func IsSoraModel(model string) bool {
	for _, m := range SoraTextToVideoModels {
		if m == model {
			return true
		}
	}
	for _, m := range SoraImageToVideoModels {
		if m == model {
			return true
		}
	}
	return false
}

// IsSoraImageToVideoModel 判断是否为 Sora 图生视频模型
func IsSoraImageToVideoModel(model string) bool {
	for _, m := range SoraImageToVideoModels {
		if m == model {
			return true
		}
	}
	return false
}

// IsVeo3Model 判断是否为 Veo3 系列模型
func IsVeo3Model(model string) bool {
	for _, m := range Veo3ModelList {
		if m == model {
			return true
		}
	}
	return false
}

// IsImageModel 判断是否为图片生成模型（NanoBanana 系列）
func IsImageModel(model string) bool {
	for _, m := range ImageModelList {
		if m == model {
			return true
		}
	}
	return false
}

// IsGpt4oImageModel 判断是否为 GPT-4o Image 模型
func IsGpt4oImageModel(model string) bool {
	for _, m := range Gpt4oImageModelList {
		if m == model {
			return true
		}
	}
	return false
}

// IsVideoModel 判断是否为视频生成模型（Sora + Veo3）
func IsVideoModel(model string) bool {
	return IsSoraModel(model) || IsVeo3Model(model)
}

// GetAllVideoModels 获取所有视频模型列表
func GetAllVideoModels() []string {
	var models []string
	models = append(models, SoraTextToVideoModels...)
	models = append(models, SoraImageToVideoModels...)
	models = append(models, Veo3ModelList...)
	return models
}

// GetAllModels 获取所有模型列表
func GetAllModels() []string {
	var models []string
	models = append(models, GetAllVideoModels()...)
	models = append(models, ImageModelList...)
	models = append(models, Gpt4oImageModelList...)
	return models
}
