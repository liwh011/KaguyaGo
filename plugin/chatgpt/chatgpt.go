package chatgpt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type chatgpt struct {
	client *openai.Client
}

func newChatgpt(key string, proxy string) *chatgpt {
	config := openai.DefaultConfig(key)
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			logrus.Fatalf("解析代理地址失败: %v", err)
		}
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	}

	c := openai.NewClientWithConfig(config)
	return &chatgpt{
		client: c,
	}
}

func (chatgpt *chatgpt) Reply(session *session) (string, error) {
	messages := []openai.ChatCompletionMessage{}

	history := session.History
	idx := 0
	for idx < len(history) {
		str := ""
		for idx < len(history) && history[idx].UserId != 114514 {
			record := history[idx]
			special := ""
			if record.IsToMe {
				special = " (向你搭话)"
			}
			str += fmt.Sprintf("%d: \"\"\"%s\"\"\"%s\n", record.UserId, record.Message, special)
			idx++
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: str,
		})
		if idx < len(history) {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: history[idx].Message,
			})
			idx++
		}
	}

	// 对message从后往前遍历，累加并统计字数，如果字数超过2000，就停止
	// 这样做的目的是为了防止一次请求的token超过API限制
	charCount := 0
	stopIdx := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if charCount+len(messages[i].Content) > 2000 {
			stopIdx = i
			// 对话必须以用户开头，所以如果停止的地方不是用户的话，就往前移动一位
			if messages[stopIdx].Role != openai.ChatMessageRoleUser {
				stopIdx++
			}
			break
		}
		charCount += len(messages[i].Content)
	}
	if stopIdx > 0 {
		messages[stopIdx].Content = "(在此之前的聊天记录被省略)" + messages[stopIdx].Content
	}
	messages = messages[stopIdx:]

	prompt := `现在有一个群聊，假设你是其中的一个成员。下面我将给出群聊中的聊天记录，格式为：ID: """消息内容"""。不同成员具有不同的ID。例如，下面是两个人的聊天：
333444555: """今天天气真好啊！"""
666874455: """你有什么打算？"""

消息内容可能包含多行文本，如果有人向你搭话，将会在对话后面特别标明。例如：
666874455: """你有什么打算？""" (向你搭话)
这代表666874455向你搭话，内容为“你有什么打算？”。你应该及时回复他的内容，否则他会觉得你文不对题。

有时候一个人的聊天会提到另一个人，通过“@ID ”的格式来表示，例如：
333444555: """今天天气真好啊！@666874455 你怎么看"""
666874455: """我觉得还行吧。"""
你可以根据其中的ID来判断他们在聊天的对象。
当你试图用ID来提及他人时，你应该使用“@ID ”的格式，请不要忘记添加ID后的空格。例如，如果你想要提及的ID为666874455，你应该使用“@666874455 ”。
而如果有人告诉你某ID对应的昵称，你可以使用昵称来提及他人。例如，如果你知道666874455的昵称为“小明”，你可以使用“小明”来提及他。

有时候由于字数限制，聊天记录旧的部分会被省略，此时对话开头会有一条提示。例如：
(在此之前的聊天记录被省略)
666874455: """哈哈哈"""
你可以当作你遗忘了前面的内容。你可以尝试推断出省略的部分，或者告诉他们你已经忘记了。

请你理解他们聊天的话题，回复一条消息来参与进话题当中。闲聊回复的长度一般为一两句话，不需要包含任何格式。下面是对你的回复的要求：
%s

以下是聊天记录，请你忘掉上面例子中的聊天场景并回复：
%s

请注意，输出中不需要包含ID和双引号，输出内容即可。比如：
反面例子：114514:"""我要去看电影。"""
正确例子：我要去看电影。
`
	messages[0].Content = fmt.Sprintf(prompt, session.Prompt, messages[0].Content)

	resp, err := chatgpt.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		},
	)
	if err != nil {
		return "", err
	}

	// logrus.Debug(messages)

	return resp.Choices[0].Message.Content, nil
}

func (chatgpt *chatgpt) ReplySingle(message string) (string, error) {
	resp, err := chatgpt.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: message,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
