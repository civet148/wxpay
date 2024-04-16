package wxpay

import "time"

const TIME_FORMAT_DATETIME = "2006-01-02 15:04:05"

// 返回n分钟前的格式化时间
func MinuteBefore(n int) string {
	return time.Now().Add(-time.Minute * time.Duration(n)).Format(TIME_FORMAT_DATETIME)
}

// 返回n分钟后的格式化时间
func MinuteAfter(n int) time.Time {
	return time.Now().Add(time.Minute * time.Duration(n))
}
