package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

var globalLogger *logrus.Logger
var logChannel chan *logrus.Entry

type ACNode struct {
	children map[rune]*ACNode
	fail     *ACNode
	isEnd    bool
	length   int
}

type AhoCorasick struct {
	root *ACNode
}

func NewAhoCorasick() *AhoCorasick {
	return &AhoCorasick{
		root: &ACNode{children: make(map[rune]*ACNode)},
	}
}

func (ac *AhoCorasick) Insert(word string) {
	node := ac.root
	for _, ch := range word {
		if _, ok := node.children[ch]; !ok {
			node.children[ch] = &ACNode{children: make(map[rune]*ACNode)}
		}
		node = node.children[ch]
	}
	node.isEnd = true
	node.length = len([]rune(word))
}

func (ac *AhoCorasick) BuildFailPointer() {
	queue := []*ACNode{ac.root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for ch, child := range current.children {
			if current == ac.root {
				child.fail = ac.root
			} else {
				fail := current.fail
				for fail != nil {
					if next, ok := fail.children[ch]; ok {
						child.fail = next
						break
					}
					fail = fail.fail
				}
				if fail == nil {
					child.fail = ac.root
				}
			}
			queue = append(queue, child)
		}
	}
}

func loadWordsIntoAC(ac *AhoCorasick, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open the sensitive words file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ac.Insert(scanner.Text())
	}

	// 构建失败指针
	ac.BuildFailPointer()

	return scanner.Err()
}

func initLoggerForToday() {
	now := time.Now()
	dateStr := now.Format("2006-01-02") // 格式化为 YYYY-MM-DD

	// 创建日志文件夹
	os.MkdirAll(filepath.Join("audit_logs", dateStr), 0755)

	globalLogger.SetOutput(&lumberjack.Logger{
		Filename:   filepath.Join("audit_logs", dateStr, "audit.log"),
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     365,  // days
		Compress:   true, // disabled by default
	})
}

func initGlobalLogger() {
	globalLogger = logrus.New()
	initLoggerForToday()

	logChannel = make(chan *logrus.Entry, 1000) // 3000是缓冲区大小，你可以根据需要调整
	// 初始化一个定时任务，每天零点重新初始化 Logger
	go func() {
		for {
			now := time.Now()
			next := now.Add(time.Hour * 24)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			ticker := time.NewTicker(next.Sub(now))
			<-ticker.C    // 阻塞，直到到达下一天的零点
			ticker.Stop() // 停止 Ticker
			initLoggerForToday()
		}
	}()
	// 另一个 goroutine 用于异步写入日志
	go func() {
		for entry := range logChannel {
			entry.Logger.Info(entry.Message)
		}
	}()
}

func auditHandler(ac *AhoCorasick) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		word := r.URL.Query().Get("word")
		bot := r.URL.Query().Get("bot")
		groupid := r.URL.Query().Get("groupid")
		friendid := r.URL.Query().Get("friendid")

		if word == "" {
			http.Error(w, "缺少 'word' 参数", http.StatusBadRequest)
			return
		}

		if bot == "" {
			http.Error(w, "缺少 'bot' 参数", http.StatusBadRequest)
			return
		}

		// 实时审计逻辑
		node := ac.root
		runes := []rune(word)
		matchedKeywords := []string{}

		for i, ch := range runes {
			for node != ac.root && node.children[ch] == nil {
				node = node.fail
			}
			if next, ok := node.children[ch]; ok {
				node = next
			}
			tmp := node
			for tmp != ac.root {
				if tmp.isEnd {
					matchedKeywords = append(matchedKeywords, string(runes[i+1-tmp.length:i+1]))
				}
				tmp = tmp.fail
			}
		}

		if len(matchedKeywords) > 0 {
			entry := globalLogger.WithFields(logrus.Fields{
				"时间":   time.Now().Format(time.RFC3339),
				"机器人":  bot,
				"群组":   groupid,
				"好友":   friendid,
				"关键词":  matchedKeywords,
				"完整消息": word,
			})

			// 将日志信息添加到日志通道
			logChannel <- entry

			// 直接输出到控制台
			fmt.Printf("审计事件：\n时间：%s\n机器人：%s\n群组：%s\n好友：%s\n关键词：%v\n完整消息：%s\n",
				time.Now().Format(time.RFC3339), bot, groupid, friendid, matchedKeywords, word)
		}

		// 返回状态码 200
		w.WriteHeader(http.StatusOK)
	}
}

func main() {
	ac := NewAhoCorasick()
	initGlobalLogger() // 初始化全局 Logger
	// 仅在服务启动时加载敏感词
	if err := loadWordsIntoAC(ac, "sensitive_words.txt"); err != nil {
		log.Fatalf("初始化敏感词库失败：%v", err)
		return
	}

	http.HandleFunc("/audit", auditHandler(ac))

	log.Println("正在监听5002端口...")
	if err := http.ListenAndServe(":5002", nil); err != nil {
		log.Fatalf("启动服务器失败：%v", err)
	}
}
