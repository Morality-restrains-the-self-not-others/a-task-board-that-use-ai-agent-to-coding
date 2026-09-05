package main

// ---- 手机号区号下拉选项（与 taskFE app/src/utils/countryDialCodes.js 保持一致） ---- //
// 展示名使用中文；code 为带 + 的 E.164 区号字符串。
// 用于公开策略接口 phone_country_options 的派生：管理员配置 allowed_phone_country_codes 后，
// 登录页下拉框只展示被允许的区域。

type countryDialOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var countryDialOptions = []countryDialOption{
	{Code: "+86", Name: "中国"},
	{Code: "+852", Name: "中国香港"},
	{Code: "+853", Name: "中国澳门"},
	{Code: "+886", Name: "中国台湾"},
	{Code: "+1", Name: "美国/加拿大"},
	{Code: "+7", Name: "俄罗斯/哈萨克斯坦"},
	{Code: "+20", Name: "埃及"},
	{Code: "+27", Name: "南非"},
	{Code: "+30", Name: "希腊"},
	{Code: "+31", Name: "荷兰"},
	{Code: "+32", Name: "比利时"},
	{Code: "+33", Name: "法国"},
	{Code: "+34", Name: "西班牙"},
	{Code: "+36", Name: "匈牙利"},
	{Code: "+39", Name: "意大利"},
	{Code: "+40", Name: "罗马尼亚"},
	{Code: "+41", Name: "瑞士"},
	{Code: "+43", Name: "奥地利"},
	{Code: "+44", Name: "英国"},
	{Code: "+45", Name: "丹麦"},
	{Code: "+46", Name: "瑞典"},
	{Code: "+47", Name: "挪威"},
	{Code: "+48", Name: "波兰"},
	{Code: "+49", Name: "德国"},
	{Code: "+51", Name: "秘鲁"},
	{Code: "+52", Name: "墨西哥"},
	{Code: "+53", Name: "古巴"},
	{Code: "+54", Name: "阿根廷"},
	{Code: "+55", Name: "巴西"},
	{Code: "+56", Name: "智利"},
	{Code: "+57", Name: "哥伦比亚"},
	{Code: "+58", Name: "委内瑞拉"},
	{Code: "+60", Name: "马来西亚"},
	{Code: "+61", Name: "澳大利亚"},
	{Code: "+62", Name: "印度尼西亚"},
	{Code: "+63", Name: "菲律宾"},
	{Code: "+64", Name: "新西兰"},
	{Code: "+65", Name: "新加坡"},
	{Code: "+66", Name: "泰国"},
	{Code: "+81", Name: "日本"},
	{Code: "+82", Name: "韩国"},
	{Code: "+84", Name: "越南"},
	{Code: "+90", Name: "土耳其"},
	{Code: "+91", Name: "印度"},
	{Code: "+92", Name: "巴基斯坦"},
	{Code: "+93", Name: "阿富汗"},
	{Code: "+94", Name: "斯里兰卡"},
	{Code: "+95", Name: "缅甸"},
	{Code: "+98", Name: "伊朗"},
	{Code: "+211", Name: "南苏丹"},
	{Code: "+212", Name: "摩洛哥"},
	{Code: "+213", Name: "阿尔及利亚"},
	{Code: "+216", Name: "突尼斯"},
	{Code: "+218", Name: "利比亚"},
	{Code: "+220", Name: "冈比亚"},
	{Code: "+221", Name: "塞内加尔"},
	{Code: "+234", Name: "尼日利亚"},
	{Code: "+254", Name: "肯尼亚"},
	{Code: "+255", Name: "坦桑尼亚"},
	{Code: "+256", Name: "乌干达"},
	{Code: "+260", Name: "赞比亚"},
	{Code: "+263", Name: "津巴布韦"},
	{Code: "+351", Name: "葡萄牙"},
	{Code: "+352", Name: "卢森堡"},
	{Code: "+353", Name: "爱尔兰"},
	{Code: "+354", Name: "冰岛"},
	{Code: "+355", Name: "阿尔巴尼亚"},
	{Code: "+356", Name: "马耳他"},
	{Code: "+357", Name: "塞浦路斯"},
	{Code: "+358", Name: "芬兰"},
	{Code: "+359", Name: "保加利亚"},
	{Code: "+370", Name: "立陶宛"},
	{Code: "+371", Name: "拉脱维亚"},
	{Code: "+372", Name: "爱沙尼亚"},
	{Code: "+380", Name: "乌克兰"},
	{Code: "+381", Name: "塞尔维亚"},
	{Code: "+385", Name: "克罗地亚"},
	{Code: "+386", Name: "斯洛文尼亚"},
	{Code: "+420", Name: "捷克"},
	{Code: "+421", Name: "斯洛伐克"},
	{Code: "+423", Name: "列支敦士登"},
	{Code: "+855", Name: "柬埔寨"},
	{Code: "+856", Name: "老挝"},
	{Code: "+880", Name: "孟加拉国"},
	{Code: "+960", Name: "马尔代夫"},
	{Code: "+961", Name: "黎巴嫩"},
	{Code: "+962", Name: "约旦"},
	{Code: "+963", Name: "叙利亚"},
	{Code: "+964", Name: "伊拉克"},
	{Code: "+965", Name: "科威特"},
	{Code: "+966", Name: "沙特阿拉伯"},
	{Code: "+967", Name: "也门"},
	{Code: "+968", Name: "阿曼"},
	{Code: "+970", Name: "巴勒斯坦"},
	{Code: "+971", Name: "阿联酋"},
	{Code: "+972", Name: "以色列"},
	{Code: "+973", Name: "巴林"},
	{Code: "+974", Name: "卡塔尔"},
	{Code: "+975", Name: "不丹"},
	{Code: "+976", Name: "蒙古"},
	{Code: "+977", Name: "尼泊尔"},
	{Code: "+992", Name: "塔吉克斯坦"},
	{Code: "+993", Name: "土库曼斯坦"},
	{Code: "+994", Name: "阿塞拜疆"},
	{Code: "+995", Name: "格鲁吉亚"},
	{Code: "+996", Name: "吉尔吉斯斯坦"},
	{Code: "+998", Name: "乌兹别克斯坦"},
}

// countryDialOptionByName 加速查找：code → option
var countryDialOptionByName = func() map[string]countryDialOption {
	m := make(map[string]countryDialOption, len(countryDialOptions))
	for _, opt := range countryDialOptions {
		m[opt.Code] = opt
	}
	return m
}()

// derivePhoneCountryOptions 从管理员配置的 allowed_phone_country_codes 派生下拉选项。
//
// - allowedCodes 为空 → 允许所有地区 → 返回空数组（前端 fallback 到全量列表）
// - allowedCodes 非空 → 只返回白名单内的选项；白名单中的未知区号也保留
//   （name 用 code 自身兜底），保证白名单严格生效，绝不含白名单外的区域。
// 返回顺序与 countryDialOptions 静态表一致，与配置顺序无关。
func derivePhoneCountryOptions(allowedCodes []string) []map[string]string {
	if len(allowedCodes) == 0 {
		return []map[string]string{}
	}
	want := make(map[string]bool, len(allowedCodes))
	for _, c := range allowedCodes {
		want[c] = true
	}
	out := make([]map[string]string, 0, len(want))
	for _, opt := range countryDialOptions {
		if want[opt.Code] {
			out = append(out, map[string]string{"code": opt.Code, "name": opt.Name})
		}
	}
	// 白名单中静态表未收录的区号（如手工写库）：保留原样，避免静默退化为"允许所有"
	for _, c := range allowedCodes {
		if _, known := countryDialOptionByName[c]; !known {
			out = append(out, map[string]string{"code": c, "name": c})
		}
	}
	return out
}
