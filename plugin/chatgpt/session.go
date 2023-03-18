package chatgpt

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/samber/lo"
)

type sessionHistoryRecord struct {
	UserId  int64
	Time    time.Time
	Message string
	IsToMe  bool
}

type session struct {
	SessionId   string
	SessionName string
	Prompt      string
	UserId      int64
	GroupId     int64
	StartTime   time.Time
	History     []sessionHistoryRecord
}

func (s *session) AddHistory(userId int64, message string, toMe bool) {
	s.History = append(s.History, sessionHistoryRecord{
		UserId:  userId,
		Time:    time.Now(),
		Message: message,
		IsToMe:  toMe,
	})
}

func (s *session) GetDialog(cnt int) string {
	dialog := ""
	for i := 0; i < cnt && i < len(s.History); i++ {
		shortMsg := strings.Replace(s.History[i].Message, "\n", " ", -1)
		runeMsg := []rune(shortMsg)
		if len(runeMsg) > 10 {
			runeMsg = runeMsg[:10]
		}
		dialog += fmt.Sprintf("%d: %s\n", s.History[i].UserId, string(runeMsg))
	}
	return dialog
}

func (s *session) GetBrief() string {
	sessionName := s.SessionName
	if sessionName == "" {
		sessionName = "未命名"
	}
	promtBrief := []rune(s.Prompt)
	if len(promtBrief) > 20 {
		promtBrief = promtBrief[:20]
	}
	return fmt.Sprintf(`【会话ID】：%s
【名称】：%s
【时间】：%s
【催眠】：%s
【摘要（共%d条）】：
%s`, s.SessionId, sessionName, s.StartTime.Format("2006-01-02 15:04:05"), string(promtBrief), len(s.History), s.GetDialog(4))
}

func (s *session) GetDetail() string {
	sessionName := s.SessionName
	if sessionName == "" {
		sessionName = "未命名"
	}
	return fmt.Sprintf(`【会话ID】：%s
【会话名称】：%s
【创建时间】：%s
【催眠咒语】：%s
【对话记录（共%d条）】：
%s`, s.SessionId, sessionName, s.StartTime.Format("2006-01-02 15:04:05"), s.Prompt, len(s.History), s.GetDialog(10))
}

type sessionManager struct {
	GroupSessions map[int64]string
	// PrivateSessions map[int64]*session
	SessionHistory []*session
}

func (sm *sessionManager) CreateGroupSession(groupId int64, prompt string) *session {
	if prompt == "" {
		prompt = `语气不用太正经，要俏皮、可爱，可以多加一些可爱的语气词，使得群聊中的所有成员都认为你是一个有趣可爱的人。
另外，群成员都喜欢说话用词带有猫娘特色、性格像猫娘的人，例如频繁地使用“喵”、“呜”等，你可以尝试使用这个语气说话。`
	}
	s := &session{
		SessionId: fmt.Sprintf("group-%d-%d", groupId, time.Now().Unix()),
		GroupId:   groupId,
		StartTime: time.Now(),
		Prompt:    prompt,
	}
	if sm.GroupSessions == nil {
		sm.GroupSessions = make(map[int64]string)
	}
	sm.GroupSessions[groupId] = s.SessionId
	sm.SessionHistory = append(sm.SessionHistory, s)
	return s
}

func (sm *sessionManager) GetGroupCurrentSession(groupId int64) *session {
	if sm.GroupSessions == nil {
		sm.GroupSessions = make(map[int64]string)
	}
	sessionId, ok := sm.GroupSessions[groupId]
	if !ok {
		return nil
	}
	session := sm.GetGroupSessionByIdOrName(groupId, sessionId)
	return session
}

func (sm *sessionManager) ResetGroupSession(groupId int64) {
	oldSession := sm.GetGroupCurrentSession(groupId)
	if oldSession != nil {
		prompt := oldSession.Prompt
		sm.CreateGroupSession(groupId, prompt)
	}
}

func (sm *sessionManager) ListGroupHistorySessions(groupId int64) []*session {
	return lo.Filter(sm.SessionHistory, func(item *session, _ int) bool {
		return item.GroupId == groupId
	})
}

func (sm *sessionManager) ListGroupHistorySessionsByPage(groupId int64, page int) ([]*session, int, int) {
	sessions := sm.ListGroupHistorySessions(groupId)
	pageSize := 5
	pageCount := (int)(math.Ceil(float64(len(sessions)) / float64(pageSize)))
	if page < 1 {
		page = 1
	}
	if page > pageCount {
		page = pageCount
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(sessions) {
		end = len(sessions)
	}
	return sessions[start:end], page, pageCount
}

func (sm *sessionManager) SwitchGroupSessionByIdOrName(groupId int64, sessionNameOrId string) {
	session := sm.GetGroupSessionByIdOrName(groupId, sessionNameOrId)
	if session != nil {
		sm.GroupSessions[groupId] = session.SessionId
	}
}

func (sm *sessionManager) SetSessionName(sessionId string, name string) {
	session := sm.GetSessionById(sessionId)
	if session != nil {
		session.SessionName = name
	}
}

func (sm *sessionManager) GetSessionByName(name string) *session {
	session, ok := lo.Find(sm.SessionHistory, func(item *session) bool {
		return item.SessionName == name
	})
	if ok {
		return session
	}
	return nil
}

func (sm *sessionManager) GetSessionById(sessionId string) *session {
	session, ok := lo.Find(sm.SessionHistory, func(item *session) bool {
		return item.SessionId == sessionId
	})
	if ok {
		return session
	}
	return nil
}

func (sm *sessionManager) GetGroupSessionByIdOrName(groupId int64, sessionNameOrId string) *session {
	session, _ := lo.Find(sm.SessionHistory, func(item *session) bool {
		return (item.SessionId == sessionNameOrId || item.SessionName == sessionNameOrId) && item.GroupId == groupId
	})
	return session
}
