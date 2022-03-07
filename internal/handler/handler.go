package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/egafa/yandexdiplom/storage"

	"time"

	"github.com/golang-jwt/jwt"
)

type tokenData struct {
	Token string `json:"token"`
}

func RegisterUser(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.AuthData{}

		log.Print("User register ", r.Body)

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			http.Error(w, "Ошибка дессериализации", http.StatusBadRequest)
			log.Print("Ошибка дессериализации ", r.Body)
			return
		}

		id, err := repo.User().Create(req)
		if err != nil {
			http.Error(w, "Ошибка дессериализации", http.StatusBadRequest)
			log.Print("Ошибка дессериализации ", r.Body)
			return
		}

		TokenTTL := time.Duration(repo.Cfg.GetInt("TokenTTL"))
		token, err := GenerateToken(id, TokenTTL, repo.Cfg.Get("SessionKey"))
		if err != nil {
			http.Error(w, "Ошибка создания токена", http.StatusBadRequest)
			log.Print("Ошибка создания токена ", err.Error())
			return
		}

		log.Print("Создан токен ", token)

		response := tokenData{}
		response.Token = token
		byt, err := json.Marshal(response)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(byt)
			w.WriteHeader(http.StatusOK)
			log.Print(" Отправлен токен " + string(byt))
			return
		}

		http.Error(w, "Ошибка отправки токена", http.StatusBadRequest)
		log.Print("Ошибка отправки токена ", err.Error())

	}
}

func LoginUser(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &storage.AuthData{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			http.Error(w, "Ошибка дессериализации", http.StatusBadRequest)
			log.Print("Ошибка дессериализации ", r.Body)
			return
		}

		id, err := repo.User().FindUser(req)
		if err != nil {
			http.Error(w, "Ошибка дессериализации", http.StatusBadRequest)
			log.Print("Ошибка дессериализации ", r.Body)
			return
		}

		TokenTTL := time.Duration(repo.Cfg.GetInt("TokenTTL"))
		token, err := GenerateToken(id, TokenTTL, repo.Cfg.Get("SessionKey"))
		if err != nil {
			http.Error(w, "Ошибка создания токена", http.StatusBadRequest)
			log.Print("Ошибка создания токена ", err.Error())
			return
		}

		response := tokenData{}
		response.Token = token
		byt, err := json.Marshal(response)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(byt)
			w.WriteHeader(http.StatusOK)
			log.Print(" Отправлен токен " + string(byt))
			return
		}

		http.Error(w, "Ошибка отправки токена", http.StatusBadRequest)
		log.Print("Ошибка отправки токена ", err.Error())

	}
}

func GetOrders(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID := r.Context().Value("userId").(*int)

		b, err := repo.GetListOrdersJSON(userID)
		if err != nil {
			http.Error(w, "Ошибка получения запросв на проверку пользователя номера заказа", http.StatusInternalServerError)
			log.Print("Ошибка получения запросв на проверку пользователя номера заказа ", err.Error())
			return
		}

		if b == nil {
			http.Error(w, "Нет данных", http.StatusNoContent)
			log.Print("Нет данных")
			return
		}

		w.Write(b)
		w.WriteHeader(http.StatusOK)
		log.Print("номера заказов успешно получены")
	}
}

func LoadOrder(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка получения номера заказа", http.StatusBadRequest)
			log.Print("Ошибка получения номера заказа ", err.Error())
			return
		}

		orderNumber := string(body)
		matched, err := regexp.MatchString(`[0-9]`, orderNumber)
		if !matched || err != nil {
			http.Error(w, "Ошибка преобразования номера заказа", http.StatusBadRequest)
			log.Print("Ошибка преобразования номера заказа ", err.Error())
			return
		}

		if !checkLuhn(orderNumber) {
			http.Error(w, "Ошибка проверки номера заказа", http.StatusBadRequest)
			log.Print("Ошибка проверки номера заказа ", err.Error())
			return
		}

		userID, ok := r.Context().Value(userCtx).(*int)
		fmt.Println(userID)

		if !ok {
			http.Error(w, "Ошибка получения ID пользователя", http.StatusBadRequest)
			log.Print("Ошибка получения ID пользователя")
			return
		}

		isNotID, err := repo.FindOrderNotID(&orderNumber, userID)
		if err != nil {
			http.Error(w, "Ошибка получения запросв на проверку другого подьзователя номера заказа", http.StatusBadRequest)
			log.Print("Ошибка получения запросв на проверку другого подьзователя номера заказа ", err.Error())
			return
		}

		if isNotID {
			http.Error(w, "номер заказа уже был загружен другим пользователем", http.StatusConflict)
			log.Print("номер заказа уже был загружен другим пользователем ", err.Error())
			return
		}

		isID, err := repo.FindOrderID(&orderNumber, userID)
		if err != nil {
			http.Error(w, "Ошибка получения запросв на проверку пользователя номера заказа", http.StatusBadRequest)
			log.Print("Ошибка получения запросв на проверку пользователя номера заказа ", err.Error())
			return
		}

		if isID {
			w.WriteHeader(http.StatusOK)
			log.Print("номер заказа уже был загружен этим пользователем ")
			return
		}

		err = repo.NewOrder(&orderNumber, userID)
		if err != nil {
			http.Error(w, "Ошибка обработки нового заказа ", http.StatusInternalServerError)
			log.Print("Ошибка обработки нового заказа ", err.Error())
			return
		}

		w.WriteHeader(http.StatusAccepted)
		log.Print("Новый номер заказа успешно обработан " + orderNumber)

	}
}

func LoadWithdraw(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка получения номера заказа", http.StatusBadRequest)
			log.Print("Ошибка получения номера заказа ", err.Error())
			return
		}

		var withdraw storage.Withdraw
		err = json.Unmarshal(body, &withdraw)

		if err != nil {
			http.Error(w, "Ошибка дессериализации", http.StatusNotImplemented)
			log.Print(" Ошибка дессериализации " + err.Error() + string(body))
			return
		}

		orderNumber := withdraw.Order
		matched, err := regexp.MatchString(`[0-9]`, orderNumber)
		if !matched || err != nil {
			http.Error(w, "Ошибка преобразования номера заказа", http.StatusUnprocessableEntity)
			log.Print("Ошибка преобразования номера заказа ", err.Error())
			return
		}

		if !checkLuhn(orderNumber) {
			http.Error(w, "Ошибка проверки номера заказа", http.StatusUnprocessableEntity)
			log.Print("Ошибка проверки номера заказа ", err.Error())
			return
		}

		userID, ok := r.Context().Value(userCtx).(*int)
		fmt.Println(userID)

		if !ok {
			http.Error(w, "Ошибка получения ID пользователя", http.StatusInternalServerError)
			log.Print("Ошибка получения ID пользователя")
			return
		}

		enough, err := repo.BalanceEnough(userID, &withdraw.Sum)
		if err != nil {
			http.Error(w, "Ошибка получения запроса баланса", http.StatusInternalServerError)
			log.Print("Ошибка получения запроса баланса", err.Error())
			return
		}

		if !enough {
			http.Error(w, "не достаточно средств", http.StatusPaymentRequired)
			log.Print("не достаточно средств")
			return
		}

		err = repo.NewWithdraw(&withdraw, userID)
		if err != nil {
			http.Error(w, "Ошибка обработки нового заказа ", http.StatusInternalServerError)
			log.Print("Ошибка обработки нового заказа ", err.Error())
			return
		}

		w.WriteHeader(http.StatusAccepted)
		log.Print("Новый номер заказа успешно обработан " + orderNumber)

	}
}

func GetListWithdraws(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID := r.Context().Value("userId").(*int)

		b, err := repo.GetListWithdrawsJSON(userID)
		if err != nil {
			http.Error(w, "Ошибка получения List Withdraws", http.StatusInternalServerError)
			log.Print("Ошибка получения List Withdraws ", err.Error())
			return
		}

		if b == nil {
			http.Error(w, "Нет данных", http.StatusNoContent)
			log.Print("Нет данных")
			return
		}

		w.Write(b)
		w.WriteHeader(http.StatusOK)
		log.Print("List Withdraws успешно получены")
	}
}

func checkLuhn(card_number string) bool { // принимаем в аргументы номер карты

	var sum int // переменная которая будет хранить проверочную сумму цифр

	ss := strings.Split(card_number, "")
	lens := len(ss)
	for i := 0; i < lens; i++ { // главный цикл, в процессе которого проверяется валидность номера карты

		number, _ := strconv.Atoi(ss[i]) // - '0';  // переводим цифру из char в int

		if i%2 == 0 { // если позиция цифры чётное, то:
			number *= 2 // умножаем цифру на 2

			if number > 9 { // согласно алгоритму, ни одно число не должно быть больше 9
				number -= 9 // второй вариант сведения к единичному разряду
			}
		}

		sum += number // прибавляем к sum номера согласно алгоритму

		if sum >= 10 { // если сумма больше либо 10
			sum -= 10 // вычитаем из суммы 10, т.к. последняя цифра не изменится
		}
	}

	return sum == 0 // вернуть, равна ли последняя цифра нулю
}

func GenerateToken(userID int, TokenTTL time.Duration, SessionKey string) (string, error) {
	userIDstr := fmt.Sprintf("%x", userID)
	tokenClaims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(TokenTTL * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
		Id:        userIDstr,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)

	return token.SignedString([]byte(SessionKey))
}
