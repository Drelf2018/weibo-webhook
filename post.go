package webhook

import (
	"database/sql"
	"sort"
	"strings"
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
	Comments   []*Post  `json:"comments"`
}

// 读取函数
func (post *Post) ReadRow(row *sql.Rows) {
	var Attachment, PicUrls, repostID string
	err := row.Scan(
		&post.Mid,
		&post.Time,
		&post.Text,
		&post.Type,
		&post.Source,

		&post.Uid,
		&post.Name,
		&post.Face,
		&post.Pendant,
		&post.Description,

		&post.Follower,
		&post.Following,

		&Attachment,
		&PicUrls,
		&repostID,
	)
	if printErr(err) {
		// 分割图片和附件
		if PicUrls == "" {
			post.PicUrls = []string{}
		} else {
			post.PicUrls = strings.Split(PicUrls, ",")
		}
		if Attachment == "" {
			post.Attachment = []string{}
		} else {
			post.Attachment = strings.Split(Attachment, ",")
		}
		post.Comments = []*Post{}

		if post.Type != "weiboComment" {
			// 添加转发的微博
			post.Repost = SavedPosts.GetPostByName(repostID)
			// 插入并排序
			SavedPosts.PushSort(*post)
		} else {
			SavedPosts.PushComment(repostID, *post)
		}
	}
}

// 判断是否在数据库
func (post *Post) Saved() bool {
	return !NewQuery("select * from posts where mid=$1 and type=$2", post.Mid, post.Type).ForEach(&Post{}, func() bool { return false })
}

// 去除空的子博文 Repost
func (post *Post) Empty() {
	if post.PicUrls == nil {
		post.PicUrls = []string{}
	}
	if post.Attachment == nil {
		post.Attachment = []string{}
	}
	if post.Comments == nil {
		post.Comments = []*Post{}
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
	} else if !monitor.In(user) {
		(&monitor).Parse(post, user)
	} else {
		return 3, "您已提交过"
	}

	if finalPost := monitor.GetPost(); finalPost != nil {
		// 推送
		go Webhook(finalPost)
		// 插入
		finalPost.Insert()
		// 清理占用
		delete(Monitors, finalPost.Type+finalPost.Mid)
	}
	return 0, "提交成功"
}

// 插入接收到的数据，包含被转发微博
//
// 我超好巧妙的递归储存
func (post *Post) Insert() string {
	if post == nil || post.Mid == "" {
		return ""
	}
	// 下个图先
	go DownloadAll(post.PicUrls, post.Face, post.Pendant)
	// 先把儿子插进去先
	repostID := post.Repost.Insert()
	// 看下自己在不在数据库 用来判断评论重复的
	if post.Saved() {
		return post.Type + post.Mid
	}

	if _, err := PostStmt.Exec(
		post.Mid,
		post.Time,
		post.Text,
		post.Type,
		post.Source,
		post.Uid,
		post.Name,
		post.Face,
		post.Pendant,
		post.Description,
		post.Follower,
		post.Following,
		strings.Join(post.Attachment, ","),
		strings.Join(post.PicUrls, ","),
		repostID,
	); !printErr(err) {
		return ""
	}

	if post.Type != "weiboComment" {
		SavedPosts.PushSort(*post)
	} else {
		SavedPosts.PushComment(repostID, *post)
	}

	return post.Type + post.Mid
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

// 不重复插入评论
func SetComments(Comments *[]*Post, post *Post) {
	for _, comment := range *Comments {
		if comment.Mid == post.Mid {
			return
		}
	}
	post.Repost = nil
	*Comments = append(*Comments, post)
}

// 插入评论
func (pl *PostList) PushComment(repostID string, post Post) {
	root := pl.GetPostByName(post.Attachment[0])
	if root == nil {
		return
	}
	if repostID == "" {
		SetComments(&root.Comments, &post)
	} else {
		commentList := root.Comments
		for i := 0; i < len(commentList); i++ {
			if repostID == post.Type+commentList[i].Mid {
				SetComments(&commentList[i].Comments, &post)
				break
			}
			commentList = append(commentList, commentList[i].Comments...)
		}
	}
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

var SavedPosts = PostList{0, 0, []Post{}, make(map[string]int)}

func init() {
	NewQuery("select * from posts order by time").ForEach(&Post{})
}
