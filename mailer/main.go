package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/smtp"
	"os"
	"sync"
	"time"
)



type Account struct {
	Email       string `json:"email"	`
	Password    string `json:"password"`
	HourlyLimit int    `json:"hourly_limit"`
	SentCount   int
	Mu          sync.Mutex 
}

type MailTask struct {
	TargetEmail string `json:"target_email"`
	Subject     string `json:"subject"`
	Content     string `json:"content"`
}



func generateHeaders(from, to, subject string) string {
    userAgents := []string{
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Thunderbird/115.0",
        "Outlook-iOS/709.2235254.prod.iphone (3.2.1)",
    }

    randomUA := userAgents[rand.Intn(len(userAgents))]

    return fmt.Sprintf("From: %s\r\n"+
        "To: %s\r\n"+
        "Subject: %s\r\n"+
        "User-Agent: %s\r\n"+ 
        "X-Mailer: Golang-Ghost-Mailer\r\n"+
        "MIME-Version: 1.0\r\n"+
        "Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n", from, to, subject, randomUA)


}





func emailWorker(id int, accounts []*Account, queue <-chan MailTask, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range queue {
		var selectedAcc *Account

		for {
			for _, acc := range accounts {
				acc.Mu.Lock()
				if acc.SentCount < acc.HourlyLimit {
					selectedAcc = acc
					acc.SentCount++
					acc.Mu.Unlock()
					goto AccountFound
				}
				acc.Mu.Unlock()
			}
			fmt.Printf("[Worker %d] Tüm hesaplarin limiti doldu, 30sn bekleniyor...\n", id)
			time.Sleep(30 * time.Second)
		}

	AccountFound:

		trackingID := rand.Intn(1000000)
		pixel := fmt.Sprintf("<br><img src='https://deadlier-prosaically-ginette.ngrok-free.dev/track?id=%d' width='1' height='1'>", trackingID)
		fullBody := generateHeaders(selectedAcc.Email, task.TargetEmail, task.Subject) + task.Content + pixel


		auth := smtp.PlainAuth("", selectedAcc.Email, selectedAcc.Password, "smtp.gmail.com")
		err := smtp.SendMail("smtp.gmail.com:587", auth, selectedAcc.Email, []string{task.TargetEmail}, []byte(fullBody))

		if err != nil {
			fmt.Printf("[Worker %d] HATA: %s -> %v\n", id, task.TargetEmail, err)
		} else {
			fmt.Printf("[Worker %d] BAŞARILI: %s (Kullanılan Hesap: %s)\n", id, task.TargetEmail, selectedAcc.Email)
		}


		delay := 5 + rand.Intn(10)
		time.Sleep(time.Duration(delay) * time.Second)
	}
}



func main() {
	rand.Seed(time.Now().UnixNano())

	dataAccounts, _ := os.ReadFile("accounts.json")
	var accounts []*Account
	json.Unmarshal(dataAccounts, &accounts)

	dataTargets, err := os.ReadFile("targets.json")
	if err != nil {
		fmt.Println("Hata: targets.json bulunamadı!")
		return
	}
	var mailler []MailTask
	json.Unmarshal(dataTargets, &mailler)

	mailQueue := make(chan MailTask, 100)
	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go emailWorker(i, accounts, mailQueue, &wg)
	}

	for _, m := range mailler {
		mailQueue <- m
	}

	close(mailQueue)
	wg.Wait()
	fmt.Println("Operasyon bitti. Tüm hedefler işlendi!")
}