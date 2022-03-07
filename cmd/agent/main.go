package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/egafa/yandexdiplom/config"
	"github.com/egafa/yandexdiplom/storage"
)

type AccuralOrder struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accural float32 `json:"Accural"`
}

func sendReq(ctx context.Context, cfg *config.ConfigAgent, repo *storage.Repo) {

	urlUpdate := "http://%s/api/orders/%s"

	for { //i := 0; i < 40; i++ {

		//time.Sleep(time.Duration(cfg.SleepInterval) * time.Second)

		select {
		case <-ctx.Done():
			return
		default:
			{
				var accuralOrder AccuralOrder

				orderDB, err := repo.FindNewOrder()

				if err != nil {
					log.Print("Не удалось сформировать запрос ", err.Error())
					continue
				}

				if orderDB.Ordernum == "" {
					log.Print("Нет новых заказов")
					continue
				}

				rtext := fmt.Sprintf(urlUpdate, cfg.AddrServer, orderDB.Ordernum)
				r, err := http.Get(rtext)

				if err != nil {
					log.Print("Не удалось сформировать запрос ", err.Error())
					continue
				}

				if r.StatusCode != http.StatusOK {
					log.Print("Не удалось сформировать запрос ")
					continue
				}

				defer r.Body.Close()

				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					log.Print(" Ошибка открытия тела ответа ", err.Error())
					continue
				}

				err = json.Unmarshal(body, &accuralOrder)

				if err != nil {
					log.Print(" Ошибка дессериализации тела ответа " + err.Error())
					continue
				}

				//accuralOrder.Order = "5246029110944032"
				//accuralOrder.Status = "PROCESSED"
				//accuralOrder.Accural = 500.00

				orderDB.Ordernum = accuralOrder.Order
				orderDB.Status = accuralOrder.Status
				orderDB.Accural = accuralOrder.Accural

				err = repo.UpdateNewOrder(&orderDB)

			}
		}
	}

}

func main() {
	cfg := config.NewAgentConfig()
	log.Println("Запуск агента.")
	log.Println("Сервер ", cfg.AddrServer)

	cfgServer := config.NewConfig()
	repo, err := storage.NewRepo(cfgServer)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer repo.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go sendReq(ctx, cfg, &repo)

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	// Block until a signal is received.
	<-sigint

	cancel()
	log.Println("Стоп агента")

}
