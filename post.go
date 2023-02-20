package webhook

import (
	"sort"
)

// 一条博文包含的信息
type Post struct {
	// 博文相关
	Mid    string `form:"mid" json:"mid"`
	Time   int64  `form:"time" json:"time"`
	Text   string `form:"text" json:"text"`
	Type   string `form:"type" json:"type"`
	Source string `form:"source" json:"source"`

	// 博主相关
	Uid         string `form:"uid" json:"uid"`
	Name        string `form:"name" json:"name"`
	Face        string `form:"face" json:"face"`
	Pendant     string `form:"pendant" json:"pendant"`
	Description string `form:"description" json:"description"`

	// 粉丝关注
	Follower  string `form:"follower" json:"follower"`
	Following string `form:"following" json:"following"`

	// 附件
	Attachment []string `form:"attachment" json:"attachment"`
	PicUrls    []string `form:"picUrls" json:"picUrls"`
	Repost     *Post    `form:"repost" json:"repost"`
}

type PostList struct {
	// 列表长度
	Length int
	// 最近博文时间
	LastPostTime int64
	// 博文列表
	Posts []Post
	// 是否存在
	Positions map[string]int
}

func (pl *PostList) Len() int {
	return pl.Length
}

func (pl *PostList) Swap(i, j int) {
	iPostID := pl.Posts[i].Type + pl.Posts[i].Mid
	jPostID := pl.Posts[j].Type + pl.Posts[j].Mid
	pl.Positions[iPostID] = j
	pl.Positions[jPostID] = i
	pl.Posts[i], pl.Posts[j] = pl.Posts[j], pl.Posts[i]
}

func (pl *PostList) Less(i, j int) bool {
	if pl.Posts[i].Time == pl.Posts[j].Time {
		return pl.Posts[i].Mid < pl.Posts[j].Mid
	}
	return pl.Posts[i].Time < pl.Posts[j].Time
}

func (pl *PostList) PushBottom(post Post) {
	pl.Length += 1
	AnyTo(post.Time > pl.LastPostTime, &pl.LastPostTime, post.Time)
	pl.Posts = append(pl.Posts, post)
	pl.Positions[post.Type+post.Mid] = pl.Length - 1
}

func (pl *PostList) PushSort(post Post) {
	pl.PushBottom(post)
	sort.Sort(pl)
}

// 根据名称返回博文
func (pl *PostList) GetPostByName(name string) *Post {
	if name == "" {
		return nil
	}
	pos, ok := pl.Positions[name]
	if ok && pos != -1 {
		return &pl.Posts[pos]
	}
	log.Errorf("博文 %v 不存在", name)
	return nil
}

// 返回给定时间之后的博文
func (pl *PostList) GetPostByTime(BeginTime, EndTime int64) []Post {
	if BeginTime > pl.LastPostTime {
		return []Post{}
	}
	end := pl.Length
	index := sort.Search(end, func(i int) bool {
		return pl.Posts[i].Time >= BeginTime
	})
	if EndTime != -1 {
		end = index + sort.Search(end-index, func(i int) bool {
			return pl.Posts[i+index].Time > EndTime
		})
	}
	return pl.Posts[index:end]
}

// 博文检查器
type PostMonitor struct {
	Score   float64
	Posts   []*Post
	Users   []*User
	Percent []float64
}

func (pm PostMonitor) Len() int {
	return len(pm.Users)
}

func (pm PostMonitor) Swap(i, j int) {
	pm.Posts[i], pm.Posts[j] = pm.Posts[j], pm.Posts[i]
	pm.Users[i], pm.Users[j] = pm.Users[j], pm.Users[i]
	pm.Percent[i], pm.Percent[j] = pm.Percent[j], pm.Percent[i]
}

func (pm PostMonitor) Less(i, j int) bool {
	return pm.Users[i].Uid < pm.Users[j].Uid
}

// 判断用户是否已经提交
//
// 参考 https://blog.csdn.net/weixin_42282999/article/details/108542734
func In(user *User, pm *PostMonitor) bool {
	sort.Sort(pm)
	index := sort.Search(pm.Len(), func(i int) bool { return pm.Users[i].Uid == user.Uid })
	return index < pm.Len()
}

var SavedPosts = PostList{0, 0, []Post{}, make(map[string]int)}
var Monitors = make(map[string]PostMonitor)

func init() {
	GetAllPost(&SavedPosts)
}

// 去除空的子博文 Repost
func (post *Post) Empty() {
	if post.PicUrls == nil {
		post.PicUrls = []string{}
	}
	if post.Attachment == nil {
		post.Attachment = []string{}
	}
	if post.Repost == nil {
		return
	}
	if post.Repost.Mid == "" {
		post.Repost = nil
	} else {
		post.Repost.Empty()
	}
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
func (post *Post) Save(user *User) (int64, string) {
	// 判断是否已推送
	pos, ok := SavedPosts.Positions[post.Type+post.Mid]
	if !ok {
		SavedPosts.Positions[post.Type+post.Mid] = -1
	} else if pos != -1 {
		return 2, "博文已经储存"
	}
	// 获取检查器
	monitor, ok := Monitors[post.Type+post.Mid]
	// 将修改后的值存回字典
	defer func() {
		Monitors[post.Type+post.Mid] = monitor
	}()

	if !ok {
		monitor = PostMonitor{1 / float64(user.Level), []*Post{post}, []*User{user}, []float64{0}}
	} else if !In(user, &monitor) {
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
		// 假如相似度为 100% 那得分只与 level 有关
		// 即一个 level 1 提交即可超过阈值而至少需要五个 level 5 提交才能超过
		monitor.Score += maxPercent / float64(user.Level)
		monitor.Posts = append(monitor.Posts, post)
		monitor.Users = append(monitor.Users, user)
		monitor.Percent = append(monitor.Percent, totPercent)
	} else {
		return 3, "您已提交过"
	}

	// 得分超过阈值，插入数据库
	if monitor.Score >= 1 {
		// 插入相似度最高的
		MaxID, i, total := 0, 0, monitor.Len()
		for i < total {
			p := monitor.Percent[i]
			AnyTo(p > monitor.Percent[MaxID], &MaxID, i)
			monitor.Users[i].Update("xp", monitor.Users[i].XP+int64(10*(p+1)/float64(total)))
			i += 1
		}

		// 发布最终确定的消息并插入数据库
		FinalPost := monitor.Posts[MaxID]
		go Webhook(FinalPost)
		FinalPost.Insert()

		// 清理占用
		delete(Monitors, FinalPost.Type+FinalPost.Mid)
	}
	return 0, "提交成功"
}
