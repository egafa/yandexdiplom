package storage

import (
	"crypto/sha1"
	"database/sql"
	"fmt"

	"github.com/egafa/yandexdiplom/config"
	"github.com/egafa/yandexdiplom/internal/model"
)

// UserRepository ...
type UserRepository struct {
	repo *Repo
}

type User struct {
	Login    string
	ID       int
	Password string
	Hash     string
}

type AuthData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r *UserRepository) Create(u *AuthData) (int, error) {

	hash := GetHash(u.Password)

	tx, err := r.repo.db.Begin()
	if err != nil {
		return 0, err
	}

	var id int
	createListQuery := fmt.Sprintf("INSERT INTO %s (login, hash) VALUES ($1, $2) RETURNING id", r.repo.UserTable)
	row := tx.QueryRow(createListQuery, u.Login, hash)
	if err := row.Scan(&id); err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, tx.Commit()

}

func (r *UserRepository) Find(id int) (*model.User, error) {
	u := &model.User{}
	selectQuery := fmt.Sprintf("SELECT id, login, hash FROM %s WHERE id = $1", r.repo.UserTable)
	if err := r.repo.db.QueryRow(selectQuery,
		id,
	).Scan(
		&u.ID,
		&u.Login,
		&u.Hash,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRecordNotFound
		}

		return nil, err
	}

	return u, nil
}

func (r *UserRepository) FindUser(u *AuthData) (int, error) {
	id := 0
	hash := GetHash(u.Password)

	selectQuery := fmt.Sprintf("SELECT id FROM %s WHERE login = $1 AND hash = $2", r.repo.UserTable)
	if err := r.repo.db.QueryRow(selectQuery,
		u.Login, hash,
	).Scan(
		&id,
	); err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrRecordNotFound
		}

		return 0, err
	}

	return id, nil
}

func GetHash(password string) string {

	hash := sha1.New()
	hash.Write([]byte(password))

	return fmt.Sprintf("%x", hash.Sum([]byte(config.GetSalt())))

}
