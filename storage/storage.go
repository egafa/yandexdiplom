package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/egafa/yandexdiplom/config"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var (
	// ErrRecordNotFound ...
	ErrRecordNotFound = errors.New("record not found")
)

type Repo struct {
	db             *sql.DB
	userRepository *UserRepository
	Cfg            *config.ConfigServer
	UserTable      string
	OrderTable     string
	WithdrawTable  string
}

func (r *Repo) User() *UserRepository {
	if r.userRepository != nil {
		return r.userRepository
	}

	r.userRepository = &UserRepository{
		repo: r,
	}

	return r.userRepository
}

func (r *Repo) Order(n string) string {

	return "hash_" + n
}

func NewRepo(cfg *config.ConfigServer) (Repo, error) {
	r := Repo{}

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		log.Println("database error ", err.Error())
		return r, err
	}

	r.db = db
	r.Cfg = cfg
	r.UserTable = cfg.UserTable
	r.OrderTable = cfg.OrderTable
	r.WithdrawTable = cfg.WithdrawTable

	db.Exec("CREATE TABLE IF NOT EXISTS " + r.UserTable +
		`("id" SERIAL PRIMARY KEY,` +
		`"login" varchar(50), "name" varchar(100), "hash" varchar(300))`)
	db.Exec("CREATE INDEX IF NOT EXISTS login ON " + r.UserTable + " (login)")

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + r.OrderTable +
		`("id" SERIAL PRIMARY KEY,` +
		`"ordernum" varchar(50), "status" varchar(20), uploaded timestamp, userid bigint, accural numeric(15,2))`)
	db.Exec("CREATE INDEX IF NOT EXISTS userid ON " + r.OrderTable + " (userid)")

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + r.WithdrawTable +
		`("id" SERIAL PRIMARY KEY,` +
		`"ordernum" varchar(50), processed timestamp, userid bigint, sum numeric(15,2))`)
	db.Exec("CREATE INDEX IF NOT EXISTS userid ON " + r.WithdrawTable + " (userid)")

	if err != nil {
		log.Println("database error ", err.Error())
		return Repo{}, err
	}

	return r, nil
}

func (r *Repo) Close() error {
	return r.db.Close()
}

type Order struct {
	Ordernum string  `json:"number"`
	UserID   int     `json:"-"`
	Status   string  `json:"status"`
	Accural  float32 `json:"accrual"`
	Uploaded string  `json:"uploaded_at"`
}

func (r *Repo) NewOrder(orderNumber *string, userID *int) error {

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtext := fmt.Sprintf("INSERT INTO %s (ordernum, userid, status, uploaded, accural) values($1,$2,$3,$4,$5)", r.OrderTable)
	stmtInsert, err := tx.Prepare(qtext)
	if err != nil {
		return err
	}

	_, err = stmtInsert.Exec(*orderNumber, *userID, "NEW", time.Now(), 0.0)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repo) FindNewOrder() (Order, error) {

	qtext := "select ordernum, userID from %s where status = $1"
	rows, err := r.db.Query(fmt.Sprintf(qtext, r.OrderTable), "NEW")
	if err != nil {
		log.Println("database error Select", rows.Err().Error())
		return Order{}, rows.Err()
	}

	defer rows.Close()

	var order Order
	if rows.Next() {

		err = rows.Scan(&order.Ordernum, &order.UserID)

		if err == nil {
			return order, nil
		}
	}
	return Order{}, err
}

func (r *Repo) UpdateNewOrder(order *Order) error {

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtext := fmt.Sprintf("select ordernum from %s where ordernum = $1", r.OrderTable)
	rows, err := r.db.Query(qtext, order.Ordernum)
	if err != nil {
		log.Println("database error Select", err.Error())
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("Не найден заказ %s для обновления", order.Ordernum)
	}

	log.Println("Обновление заказа ", order.Ordernum, " статус ", order.Status, "Accural ", order.Accural)

	qtext = "UPDATE %s SET status = $1, accural = $2 WHERE ordernum=$3"
	_, err = tx.Exec(fmt.Sprintf(qtext, r.OrderTable), order.Status, order.Accural, order.Ordernum)

	return tx.Commit()

}

func (r *Repo) FindOrderNotID(orderNumber *string, userID *int) (bool, error) {

	qtext := fmt.Sprintf("select ordernum, user from %s where (ordernum='%s') and (userid!=$1)", r.OrderTable, *orderNumber)
	//rows, err := r.db.Query(qtext, fmt.Sprintf("%x", *userID))
	rows, err := r.db.Query(qtext, *userID)
	if rows.Err() != nil {
		log.Println("database error Select", err.Error())
		return false, err
	}

	defer rows.Close()

	if rows.Next() {
		return true, nil
	}

	return false, nil
}

func (r *Repo) FindOrderID(orderNumber *string, userID *int) (bool, error) {

	qtext := fmt.Sprintf("select ordernum, user from %s where (ordernum='%s') and (userid=$1)", r.OrderTable, *orderNumber)
	rows, err := r.db.Query(qtext, *userID)
	if rows.Err() != nil {
		log.Println("database error Select", err.Error())
		return false, err
	}

	defer rows.Close()

	if rows.Next() {
		return true, nil
	}

	return false, nil
}

func (r *Repo) GetListOrdersJSON(userID *int) ([]byte, error) {
	m, err := r.GetListOrders(userID)

	if len(m) == 0 {
		return nil, nil
	}

	res, err := json.MarshalIndent(m, "", "")
	if err != nil {
		return nil, err
	}
	return res, nil

}

func (r *Repo) GetListOrders(userID *int) ([]Order, error) {
	var res []Order

	qtext := fmt.Sprintf("select ordernum, status, accural, uploaded from %v where userID = $1", r.OrderTable)
	rows, err := r.db.Query(qtext, *userID)
	if err != nil {
		log.Println("database error Select", err.Error())
		return res, err
	}
	defer rows.Close()

	for {
		if !rows.Next() {
			break
		}
		var order Order
		err = rows.Scan(&order.Ordernum, &order.Status, &order.Accural, &order.Uploaded)
		if err == nil {
			res = append(res, order)
		} else {
			return res, err
		}
	}

	return res, nil
}

type Withdraw struct {
	Order     string  `json:"order"`
	Sum       float32 `json:"sum"`
	Processed string  `json:"processed_at"`
}

func (r *Repo) GetListWithdrawsJSON(userID *int) ([]byte, error) {
	m, err := r.GetListWithdraws(userID)

	if len(m) == 0 {
		return nil, nil
	}

	res, err := json.MarshalIndent(m, "", "")
	if err != nil {
		return nil, err
	}
	return res, nil

}

func (r *Repo) GetListWithdraws(userID *int) ([]Withdraw, error) {
	var res []Withdraw

	qtext := fmt.Sprintf("select ordernum,  sum, processed from %v where userID = $1", r.WithdrawTable)
	rows, err := r.db.Query(qtext, *userID)
	if err != nil {
		log.Println("database error Select", err.Error())
		return res, err
	}
	defer rows.Close()

	for {
		if !rows.Next() {
			break
		}
		var withdraw Withdraw
		err = rows.Scan(&withdraw.Order, &withdraw.Sum, &withdraw.Processed)
		if err == nil {
			res = append(res, withdraw)
		} else {
			return nil, err
		}
	}

	return res, nil
}

func (r *Repo) NewWithdraw(w *Withdraw, userID *int) error {

	qtext := "select ordernum from %s where ordernum = $1"
	rows, err := r.db.Query(fmt.Sprintf(qtext, r.WithdrawTable), w.Order)
	if err != nil {
		log.Println("database error Select", rows.Err().Error())
		return err
	}
	defer rows.Close()

	if rows.Next() {
		return fmt.Errorf("Уже есть номер заказа %s", w.Order)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtext = fmt.Sprintf("INSERT INTO %s (ordernum, userid, processed, sum) values($1,$2,$3,$4)", r.WithdrawTable)
	stmtInsert, err := tx.Prepare(qtext)
	if err != nil {
		return err
	}

	_, err = stmtInsert.Exec(w.Order, *userID, time.Now(), w.Sum)

	log.Print("Добавление записи в ", r.WithdrawTable, *w)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repo) BalanceEnough(userID *int, sum *float32) (bool, error) {
	b, err := r.GetBalance(userID)

	if err != nil {
		log.Println("database error Select", err.Error())
		return false, err
	}

	if (b.Accural - b.Withdrawn) >= *sum {
		return true, nil
	}

	return false, nil
}

type Balance struct {
	Accural   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

func (r *Repo) GetBalanceJSON(userID *int) ([]byte, error) {
	b, err := r.GetBalance(userID)

	res, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *Repo) GetBalance(userID *int) (Balance, error) {
	var res Balance

	qtext := fmt.Sprintf("select accural from %v where userID = $1", r.OrderTable)
	rows, err := r.db.Query(qtext, *userID)
	if err != nil {
		log.Println("database error Select", err.Error())
		return Balance{}, err
	}
	defer rows.Close()

	sum, err := sumRows(rows)
	if err != nil {
		log.Println("database error Select", err.Error())
		return Balance{}, err
	}

	res.Accural = sum
	sum = 0

	qtext = fmt.Sprintf("select sum from %v where userID = $1", r.WithdrawTable)
	rows, err = r.db.Query(qtext, *userID)
	if err != nil {
		log.Println("database error Select", err.Error())
		return Balance{}, err
	}
	defer rows.Close()

	sum, err = sumRows(rows)
	if err != nil {
		log.Println("database error Select", err.Error())
		return Balance{}, err
	}

	res.Withdrawn = sum

	return res, nil
}

func sumRows(rows *sql.Rows) (float32, error) {
	var sum float32
	for {
		if !rows.Next() {
			break
		}

		var s float32
		err := rows.Scan(&s)
		if err == nil {
			sum = sum + s
		} else {
			return 0, err
		}
	}

	return sum, nil
}
