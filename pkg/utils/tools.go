/*
 * @Date: 2026-06-15 17:54:12
 * @LastEditTime: 2026-06-15 17:54:13
 * @FilePath: /dark_pkg/pkg/utils/tools.go
 * @Description:
 */
package utils

import (
	"strings"
	"unicode"
)

/**
 * @description: 转换为驼峰式
 * @param {string} s
 * @return {*}
 */
func ToCamelCase(s string) string {
	var res strings.Builder
	words := strings.Split(s, "_")
	for _, w := range words {
		if w == "" {
			continue
		}
		runes := []rune(w)
		runes[0] = unicode.ToUpper(runes[0])
		res.WriteString(string(runes))
	}
	return res.String()
}

/**
 * @description: 转换为驼峰式-小写
 * @param {string} s
 * @return {*}
 */
func ToLowerCamel(s string) string {
	camel := ToCamelCase(s)
	if camel == "" {
		return ""
	}
	runes := []rune(camel)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
