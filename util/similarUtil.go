package util

import "strings"

// 计算两个字符串的编辑距离
func calculateEditDistance(str1, str2 string) int {
	len1 := len(str1)
	len2 := len(str2)

	// 创建二维数组用于存储编辑距离
	dp := make([][]int, len1+1)
	for i := range dp {
		dp[i] = make([]int, len2+1)
	}

	// 初始化边界条件
	for i := 0; i <= len1; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		dp[0][j] = j
	}

	// 计算编辑距离
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			if str1[i-1] == str2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = min(dp[i-1][j-1]+1, min(dp[i][j-1]+1, dp[i-1][j]+1))
			}
		}
	}

	return dp[len1][len2]
}

// 求最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EvaluateSimilar 评估两个字符串的相似度
func EvaluateSimilar(str1, str2 string) bool {
	str1 = strings.ToLower(strings.TrimSpace(str1))
	str2 = strings.ToLower(strings.TrimSpace(str2))
	editDistance := calculateEditDistance(str1, str2)
	similarity := 1 - float64(editDistance)/float64(max(len(str1), len(str2)))
	return similarity > 0.8
}
