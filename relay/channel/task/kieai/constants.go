package kieai

const ChannelName = "KieAI"

// 视频模型列表
var VideoModelList = []string{
	"sora-2-pro",
}

// 图片模型列表
var ImageModelList = []string{
	"nano-banana-pro",
	"nano-banana",
}

// 模型名称映射：对外模型名 -> Kie.ai 内部模型名
var ModelNameMapping = map[string]string{
	"sora-2-pro":      "sora-2-pro-text-to-video",
	"nano-banana-pro": "nano-banana-pro",
	"nano-banana":     "nano-banana",
}

// GetKieModelName 获取 Kie.ai 内部模型名称
func GetKieModelName(externalModel string) string {
	if kieModel, ok := ModelNameMapping[externalModel]; ok {
		return kieModel
	}
	return externalModel
}

// IsVideoModel 判断是否为视频模型
func IsVideoModel(model string) bool {
	for _, m := range VideoModelList {
		if m == model {
			return true
		}
	}
	return false
}

// IsImageModel 判断是否为图片模型
func IsImageModel(model string) bool {
	for _, m := range ImageModelList {
		if m == model {
			return true
		}
	}
	return false
}

