package webhook

import "sort"

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
func (pm *PostMonitor) In(user *User) bool {
	sort.Sort(pm)
	index := sort.Search(pm.Len(), func(i int) bool { return pm.Users[i].Uid == user.Uid })
	return index < pm.Len()
}

func (monitor *PostMonitor) Parse(post *Post, user *User) {
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
}

func (monitor PostMonitor) GetPost() *Post {
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

		// 发布最终确定的博文
		return monitor.Posts[MaxID]
	}
	return nil
}

var Monitors = make(map[string]PostMonitor)
