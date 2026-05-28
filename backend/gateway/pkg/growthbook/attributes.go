package growthbook

import "fmt"

// NewAttributes 创建用户属性
func NewAttributes(userID int64, userRole int) *Attributes {
	return &Attributes{
		ID:   fmt.Sprintf("%d", userID),
		Role: userRole,
	}
}

// WithEmail 设置邮箱属性
func (a *Attributes) WithEmail(email string) *Attributes {
	a.Email = email
	return a
}

// WithDeviceType 设置设备类型属性
func (a *Attributes) WithDeviceType(deviceType string) *Attributes {
	a.DeviceType = deviceType
	return a
}

// WithBrowser 设置浏览器属性
func (a *Attributes) WithBrowser(browser string) *Attributes {
	a.Browser = browser
	return a
}

// WithCustom 设置自定义属性
func (a *Attributes) WithCustom(key string, value interface{}) *Attributes {
	if a.Custom == nil {
		a.Custom = make(map[string]interface{})
	}
	a.Custom[key] = value
	return a
}

// ToMap 转换为 map 格式（用于 API 传输）
func (a *Attributes) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"id":   a.ID,
		"role": a.Role,
	}

	if a.Email != "" {
		result["email"] = a.Email
	}
	if a.DeviceType != "" {
		result["deviceType"] = a.DeviceType
	}
	if a.Browser != "" {
		result["browser"] = a.Browser
	}

	for k, v := range a.Custom {
		result[k] = v
	}

	return result
}

// Builder 属性构建器
type Builder struct {
	attrs *Attributes
}

// NewBuilder 创建属性构建器
func NewBuilder(userID int64, userRole int) *Builder {
	return &Builder{
		attrs: NewAttributes(userID, userRole),
	}
}

// Email 设置邮箱
func (b *Builder) Email(email string) *Builder {
	b.attrs.Email = email
	return b
}

// DeviceType 设置设备类型
func (b *Builder) DeviceType(deviceType string) *Builder {
	b.attrs.DeviceType = deviceType
	return b
}

// Browser 设置浏览器
func (b *Builder) Browser(browser string) *Builder {
	b.attrs.Browser = browser
	return b
}

// Custom 设置自定义属性
func (b *Builder) Custom(key string, value interface{}) *Builder {
	b.attrs.WithCustom(key, value)
	return b
}

// Build 构建最终属性
func (b *Builder) Build() *Attributes {
	return b.attrs
}