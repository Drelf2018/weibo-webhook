package main

// 一条博文包含的信息
type Post struct {
	Mid    string  `form:"mid" json:"mid"`
	Time   float64 `form:"time" json:"time"`
	Text   string  `form:"text" json:"text"`
	Type   string  `form:"type" json:"type"`
	Source string  `form:"source" json:"source"`

	Uid      string `form:"uid" json:"uid"`
	Name     string `form:"name" json:"name"`
	Face     string `form:"face" json:"face"`
	Follow   string `form:"follow" json:"follow"`
	Follower string `form:"follower" json:"follower"`
	Desc     string `form:"description" json:"description"`

	PicUrls []string `form:"picUrls" json:"picUrls"`
	Repost  *Post    `form:"repost,omitempty" json:"repost"`
}

// 博文检查器
type PostMonitor struct {
	Score    float64
	Posts    []Post
	Percent  []float64
	isPosted bool
}

var PostList []Post
var Monitors = make(map[string]PostMonitor)

func init() {
	PostList = GetAllPost()
	for _, post := range PostList {
		Monitors[post.Type+post.Mid] = PostMonitor{1, nil, nil, true}
	}
}

// 返回给定时间之后的博文
func GetPostByTime(BeginTime float64) []Post {
	return Filter(PostList, func(p Post) bool {
		return (p.Time >= BeginTime)
	})
}

// 在 PostList 中添加博文并插入数据库
//
// 返回值
//
// 0: 提交成功
//
// 1: token 错误
//
// 2: 博文已被储存
func (post *Post) Save(token string) (int64, string) {
	// 提交者级别
	level := GetLevelByToken(token)
	if level <= 0 {
		return 1, "token 错误"
	}
	log.Infof("收到 token: %v(LV%v) 博文：%v", token, level, post.Text)
	// 初始化检查器
	monitor, ok := Monitors[post.Type+post.Mid]
	// 将修改后的值存回字典
	defer func() {
		Monitors[post.Type+post.Mid] = monitor
	}()

	if !ok {
		monitor = PostMonitor{1 / level, []Post{*post}, []float64{0}, false}
	} else if monitor.isPosted {
		return 2, "博文已被储存"
	} else {
		// 检测提交的文本与检查器中储存的文本的相似性
		maxPercent := 0.0
		totPercent := 0.0
		for i, p := range monitor.Posts {
			percent := SimilarText(p.Text, post.Text)
			if percent > maxPercent {
				maxPercent = percent
			}
			totPercent += percent
			monitor.Percent[i] += percent
		}
		// 更新可信度得分
		// 加入相似度为 100% 那得分只与 level 有关
		// 即一个 level 1 提交即可超过阈值而至少需要五个 level 5 提交才能超过
		monitor.Score += (1 / level) * maxPercent
		monitor.Posts = append(monitor.Posts, *post)
		monitor.Percent = append(monitor.Percent, totPercent)
	}

	// 得分超过阈值，插入数据库
	if monitor.Score >= 1 {
		// 插入相似度最高的
		MaxID := 0
		for i, p := range monitor.Percent {
			AnyTo(p > monitor.Percent[MaxID], &MaxID, i)
		}

		// 发布最终确定的消息并插入数据库
		FinalPost := monitor.Posts[MaxID]
		go Webhook(FinalPost)
		FinalPost.Insert()

		// 清理占用，但要留下 isPosted 避免重复提交
		monitor.isPosted = true
		monitor.Percent = nil
		monitor.Posts = nil
	}
	return 0, "提交成功"
}
