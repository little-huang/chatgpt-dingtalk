package chatgpt

import (
	"fmt"
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/eryajf/chatgpt-dingtalk/public"
	"github.com/eryajf/chatgpt-dingtalk/pkg/logger"
	openai "github.com/sashabaranov/go-openai"
)

type ChatGPT struct {
	client         *openai.Client
	ctx            context.Context
	userId         string
	maxQuestionLen int
	maxText        int
	maxAnswerLen   int
	timeOut        time.Duration // 超时时间, 0表示不超时
	doneChan       chan struct{}
	cancel         func()

	ChatContext *ChatContext
}

func New(userId string) *ChatGPT {
	var ctx context.Context
	var cancel func()

	ctx, cancel = context.WithTimeout(context.Background(), 600*time.Second)
	timeOutChan := make(chan struct{}, 1)
	go func() {
		<-ctx.Done()
		timeOutChan <- struct{}{} // 发送超时信号，或是提示结束，用于聊天机器人场景，配合GetTimeOutChan() 使用
	}()
  
	logger.Info(fmt.Sprintf("public.config %v", public.Config))
  
	config := openai.DefaultConfig(public.Config.ApiKey)
	if public.Config.AzureOn {
		config = openai.DefaultAzureConfig(
			public.Config.AzureOpenAIToken,
			"https://"+public.Config.AzureResourceName+".openai."+
				"azure.com/",
			public.Config.AzureDeploymentName,
		)
    
		logger.Info(fmt.Sprintf(" DefaultAzureConfig %v", config))

	} else {
		if public.Config.HttpProxy != "" {
			config.HTTPClient.Transport = &http.Transport{
				// 设置代理
				Proxy: func(req *http.Request) (*url.URL, error) {
					return url.Parse(public.Config.HttpProxy)
				}}
		}
		if public.Config.BaseURL != "" {
			config.BaseURL = public.Config.BaseURL + "/v1"
		}
	}

	return &ChatGPT{
		client:         openai.NewClientWithConfig(config),
		ctx:            ctx,
		userId:         userId,
		maxQuestionLen: 2048, // 最大问题长度
		maxAnswerLen:   2048, // 最大答案长度
		maxText:        4096, // 最大文本 = 问题 + 回答, 接口限制
		timeOut:        public.Config.SessionTimeout,
		doneChan:       timeOutChan,
		cancel: func() {
			cancel()
		},
		ChatContext: NewContext(),
	}
}
func (c *ChatGPT) Close() {
	c.cancel()
}

func (c *ChatGPT) GetDoneChan() chan struct{} {
	return c.doneChan
}

func (c *ChatGPT) SetMaxQuestionLen(maxQuestionLen int) int {
	if maxQuestionLen > c.maxText-c.maxAnswerLen {
		maxQuestionLen = c.maxText - c.maxAnswerLen
	}
	c.maxQuestionLen = maxQuestionLen
	return c.maxQuestionLen
}
