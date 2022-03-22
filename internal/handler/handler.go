package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"

	"github.com/ShiraazMoollatjie/goluhn"
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

		//log.Print("User register ", r.Body)

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

		TokenTTL := time.Duration(repo.Cfg.TokenTTL)
		token, err := GenerateToken(id, TokenTTL, repo.Cfg.SessionKey)
		if err != nil {
			http.Error(w, "Ошибка создания токена", http.StatusBadRequest)
			log.Print("Ошибка создания токена ", err.Error())
			return
		}

		//log.Print("Создан токен ", token)

		response := tokenData{}
		response.Token = token
		byt, err := json.Marshal(response)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Authorization", "Bearer "+token)
			w.Write(byt)
			w.WriteHeader(http.StatusOK)
			//log.Print(" Отправлен токен " + string(byt))
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

		TokenTTL := time.Duration(repo.Cfg.TokenTTL)
		token, err := GenerateToken(id, TokenTTL, repo.Cfg.SessionKey)
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
			w.Header().Set("Authorization", "Bearer "+token)
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

		logText := "********* GetOrders ********** "
		log.Print(logText, r.RequestURI)

		userID := r.Context().Value(userCtx).(*int)

		b, err := repo.GetListOrdersJSON(userID)
		if err != nil {
			http.Error(w, "Ошибка получения запроса на проверку пользователя номера заказа", http.StatusInternalServerError)
			log.Print("Ошибка получения запроса на проверку пользователя номера заказа ", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if b == nil {
			w.WriteHeader(http.StatusNoContent)
			//http.Error(w, "Нет данных", http.StatusNoContent)
			log.Print(logText, " Нет данных у пользователя ", *userID)
			return
		}

		w.Write(b)
		w.WriteHeader(http.StatusOK)
		log.Print(logText, " номера заказов успешно получены ", *userID)
	}
}

func GetBalance(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		userID := r.Context().Value("userId").(*int)

		logText := "********* GetBalance ********** "
		log.Print(logText, r.RequestURI, " userID ", *userID)

		b, err := repo.GetBalanceJSON(userID)
		if err != nil {
			http.Error(w, "Ошибка баланса запросв на проверку пользователя номера заказа", http.StatusInternalServerError)
			log.Print("Ошибка получения запросв на проверку пользователя номера заказа ", err.Error())
			return
		}

		if b == nil {
			http.Error(w, "Нет данных", http.StatusNoContent)
			log.Print("Нет данных")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
		w.WriteHeader(http.StatusOK)
		log.Print(logText, " номера заказов успешно получены "+string(b))

	}
}

func LoadOrder(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		logText := "********* Load order ********** "
		log.Print(logText, r.RequestURI)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка получения номера заказа", http.StatusBadRequest)
			log.Print(logText+"Ошибка получения номера заказа ", err.Error())
			return
		}

		orderNumber := string(body)
		fmt.Println(logText+" orderNumber = ", orderNumber)

		matched, err := regexp.MatchString(`[0-9]`, orderNumber)
		if err != nil {
			http.Error(w, "Ошибка проверки на вхождение цифр номера заказа", http.StatusBadRequest)
			log.Print(logText+"Ошибка проверки на вхождение цифр номера заказа", err.Error())
			return
		}
		if !matched {
			http.Error(w, "Ошибка проверки на вхождение цифр номера заказа", http.StatusUnprocessableEntity)
			log.Print(logText + "Ошибка проверки на вхождение цифр номера заказа ")
			return
		}

		err = goluhn.Validate(orderNumber)
		if err != nil {
			http.Error(w, "Ошибка проверки номера заказа goluhn", http.StatusUnprocessableEntity)
			log.Print(logText+"Ошибка проверки номера заказа goluhn ", orderNumber)
			return
		}

		fmt.Println(logText + " Получение USER ID")

		userID, ok := r.Context().Value(userCtx).(*int)

		if !ok {
			http.Error(w, "Ошибка получения ID пользователя", http.StatusBadRequest)
			log.Print(logText + "Ошибка получения ID пользователя")
			return
		}
		fmt.Println(logText+" USER ID = ", *userID)

		isNotID, err := repo.FindOrderNotID(&orderNumber, userID)
		if err != nil {
			http.Error(w, "Ошибка получения запросв на проверку другого подьзователя номера заказа", http.StatusConflict)
			log.Print(logText+"Ошибка получения запросв на проверку другого подьзователя номера заказа ", err.Error())
			return
		}

		if isNotID {
			w.WriteHeader(http.StatusConflict)
			//http.Error(w, "номер заказа уже был загружен другим пользователем", http.StatusConflict)
			log.Print(logText + "номер заказа уже был загружен другим пользователем ")
			return
		}

		isID, err := repo.FindOrderID(&orderNumber, userID)
		if err != nil {
			http.Error(w, "Ошибка получения запросв на проверку пользователя номера заказа", http.StatusConflict)
			log.Print(logText+"Ошибка получения запросв на проверку пользователя номера заказа ", err.Error())
			return
		}

		if isID {
			w.WriteHeader(http.StatusOK)
			log.Print(logText + "номер заказа уже был загружен этим пользователем")
			return
		}

		err = repo.NewOrder(&orderNumber, userID)
		if err != nil {
			http.Error(w, "Ошибка обработки нового заказа ", http.StatusInternalServerError)
			log.Print(logText+"Ошибка обработки нового заказа ", err.Error())
			return
		}

		w.WriteHeader(http.StatusAccepted)
		log.Print(logText + "Новый номер заказа успешно обработан " + orderNumber)

	}
}

func LoadWithdraw(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		defer r.Body.Close()

		logText := "******** LoadWithdraw " + r.RequestURI

		userID, ok := r.Context().Value(userCtx).(*int)
		if !ok {
			http.Error(w, "Ошибка получения ID пользователя", http.StatusInternalServerError)
			log.Print(logText, "Ошибка получения ID пользователя")
			return
		}

		log.Print(logText, " userID ", *userID)

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
			log.Print(logText, " Ошибка преобразования номера заказа ")
			return
		}

		err = goluhn.Validate(orderNumber)
		if err != nil {
			http.Error(w, "Ошибка проверки номера заказа goluhn", http.StatusUnprocessableEntity)
			log.Print(logText+"Ошибка проверки номера заказа goluhn ", orderNumber)
			return
		}

		log.Print(logText, withdraw)

		enough, err := repo.BalanceEnough(userID, &withdraw.Sum)
		if err != nil {
			http.Error(w, "Ошибка получения запроса баланса", http.StatusInternalServerError)
			log.Print("Ошибка получения запроса баланса", err.Error())
			return
		}

		if !enough {
			http.Error(w, "не достаточно средств ", http.StatusPaymentRequired)
			log.Print("не достаточно средств ", withdraw)
			return
		}

		err = repo.NewWithdraw(&withdraw, userID)
		if err != nil {
			http.Error(w, "Ошибка обработки нового заказа ", http.StatusInternalServerError)
			log.Print("Ошибка обработки нового заказа ", err.Error())
			return
		}

		w.WriteHeader(http.StatusAccepted)
		log.Print(logText, " Новый номер заказа успешно обработан "+orderNumber)

	}
}

func GetListWithdraws(repo *storage.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		logText := "******** GetListWithdraws "

		userID, ok := r.Context().Value(userCtx).(*int)
		if !ok {
			http.Error(w, "Ошибка получения ID пользователя", http.StatusInternalServerError)
			log.Print(logText, "Ошибка получения ID пользователя")
			return
		}

		log.Print(logText, r.RequestURI, " userID ", userID)

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
		w.Header().Set("Content-Type", "application/json")
		log.Print("List Withdraws успешно получены")
	}
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
