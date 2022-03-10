package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/egafa/yandexdiplom/config"
	"github.com/egafa/yandexdiplom/storage"
)

type AccuralOrder struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accural float32 `json:"Accural"`
}

func sendReq(ctx context.Context, cfg *config.ConfigServer, repo *storage.Repo) {

	urlUpdate := "http://%s/api/orders/%s"

	client := &http.Client{}

	for { //i := 0; i < 40; i++ {

		time.Sleep(time.Duration(cfg.SleepInterval) * time.Second)

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

				raddr := fmt.Sprintf(urlUpdate, cfg.AccuralAddress, orderDB.Ordernum)
				r, err := http.NewRequest(http.MethodPost, raddr, nil)
				if err != nil {
					log.Print("Не удалось сформировать запрос получения данных заказа ", err.Error())
					continue
				}
				r.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(r)
				if err != nil {
					log.Print("Ошибка выполнения запроса получения данных заказа ", err.Error())
					continue
				}

				if resp.StatusCode != http.StatusOK {
					log.Print("Ошибочный код выполнения запроса получения данных заказа")
					continue
				}

				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Print(" Ошибка открытия тела ответа запроса получения данных заказа", err.Error())
					continue
				}

				err = json.Unmarshal(body, &accuralOrder)

				if err != nil {
					log.Print("Ошибка дессериализации тела ответа " + err.Error())
					continue
				}

				log.Print("Отправлен запрос получения данных заказа ", raddr, orderDB)

				//accuralOrder.Order = "5246029110944032"
				//accuralOrder.Status = "PROCESSED"
				//accuralOrder.Accural = 500.00

				orderDB.Ordernum = accuralOrder.Order
				orderDB.Status = accuralOrder.Status
				orderDB.Accural = accuralOrder.Accural

				err = repo.UpdateNewOrder(&orderDB)
				if err != nil {
					log.Print("Ошибка обновления данных заказа ", err.Error())
					continue
				}

			}
		}
	}

}
