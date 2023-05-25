package webhook

import (
	"os"
	"runtime"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

// 全局 log
var log = &logrus.Logger{
	Out: os.Stderr,
	Formatter: &nested.Formatter{
		HideKeys:        true,
		TimestampFormat: time.Kitchen,
	},
	Hooks: make(logrus.LevelHooks),
	Level: logrus.DebugLevel,
}

// 过滤函数 时间复杂度暂且不说
func Filter[T any](s []T, fn func(T) bool) []T {
	result := make([]T, 0, len(s))
	for _, v := range s {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

// 三目运算符
func Any[T any](Expr bool, TrueReturn, FalseReturn T) T {
	if Expr {
		return TrueReturn
	}
	return FalseReturn
}

// 直接赋值的三目运算符
func AnyTo[T any](Expr bool, Pointer *T, Value T) {
	if Expr {
		*Pointer = Value
	}
}

// 简易的 panic
func panicErr(err error) bool {
	if err != nil {
		panic(err)
	} else {
		return true
	}
}

// 判断第二个变量
func panicSecond[T any](res T, err error) T {
	panicErr(err)
	return res
}

// 如果出错则打印错误并返回 false 否则返回 true
func printErr(err error) bool {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if ok {
			log.Errorf("%v occured in %v:%v", err, file, line)
		} else {
			log.Error(err)
		}
		return false
	} else {
		return true
	}
}

// 文本相似性判断
//
// 参考 https://blog.csdn.net/weixin_30402085/article/details/96165537
func SimilarText(first, second string) float64 {
	var similarText func(string, string, int, int) int
	similarText = func(str1, str2 string, len1, len2 int) int {
		var sum, max int
		pos1, pos2 := 0, 0

		// Find the longest segment of the same section in two strings
		for i := 0; i < len1; i++ {
			for j := 0; j < len2; j++ {
				for l := 0; (i+l < len1) && (j+l < len2) && (str1[i+l] == str2[j+l]); l++ {
					if l+1 > max {
						max = l + 1
						pos1 = i
						pos2 = j
					}
				}
			}
		}

		if sum = max; sum > 0 {
			if pos1 > 0 && pos2 > 0 {
				sum += similarText(str1, str2, pos1, pos2)
			}
			if (pos1+max < len1) && (pos2+max < len2) {
				s1 := []byte(str1)
				s2 := []byte(str2)
				sum += similarText(string(s1[pos1+max:]), string(s2[pos2+max:]), len1-pos1-max, len2-pos2-max)
			}
		}

		return sum
	}

	l1, l2 := len(first), len(second)
	if l1+l2 == 0 {
		return 0
	}
	sim := similarText(first, second, l1, l2)
	return float64(sim*2) / float64(l1+l2)
}

// 字符串前后去指定字符
//
// 参考: https://blog.csdn.net/a1053904672/article/details/90758573
func RemoveChars(s_ string, chars_ string) string {
	s, chars := []rune(s_), []rune(chars_)
	length := len(s)
	max := len(s) - 1
	l, r := true, true //标记当左端或者右端找到正常字符后就停止继续寻找
	start, end := 0, max
	tmpEnd := 0
	charset := make(map[rune]bool) //创建字符集，也就是唯一的字符，方便后面判断是否存在
	for i := 0; i < len(chars); i++ {
		charset[chars[i]] = true
	}
	for i := 0; i < length; i++ {
		if _, exist := charset[s[i]]; l && !exist {
			start = i
			l = false
		}
		tmpEnd = max - i
		if _, exist := charset[s[tmpEnd]]; r && !exist {
			end = tmpEnd
			r = false
		}
		if !l && !r {
			break
		}
	}
	if l && r { // 如果左端和右端都没找到正常字符，那么表示该字符串没有正常字符
		return ""
	}
	return string(s[start : end+1])
}

func Strip(s string) string {
	return RemoveChars(s, " \n")
}
