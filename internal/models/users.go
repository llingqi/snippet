package models

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

// Define a new User type. Notice how the field names and types align
// with the columns in the database "users" table?
type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
}
type UserModelInterface interface {
	Insert(name, email, password string) error
	Authenticate(email, password string) (int, error)
	Exists(id int) (bool, error)
	Get(id int) (*User, error)
	PasswordUpdate(id int, currentPassword, newPassword string) error
}

// Define a new UserModel type which wraps a database connection pool.
type UserModel struct {
	DB *sql.DB
}

// We'll use the Insert method to add a new record to the "users" table.
func (m *UserModel) Insert(name, email, password string) error {
	// Create a bcrypt hash of the plain-text password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO users (name, email, hashed_password, created)
    VALUES(?, ?, ?, UTC_TIMESTAMP())`

	// Use the Exec() method to insert the user details and hashed password
	// into the users table.
	_, err = m.DB.Exec(stmt, name, email, string(hashedPassword))
	if err != nil {
		// If this returns an error, we use the errors.As() function to check
		// whether the error has the type *mysql.MySQLError. If it does, the
		// error will be assigned to the mySQLError variable. We can then check
		// whether or not the error relates to our users_uc_email key by
		// checking if the error code equals 1062 and the contents of the error
		// message string. If it does, we return an ErrDuplicateEmail error.
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			if mySQLError.Number == 1062 && strings.Contains(mySQLError.Message, "users_uc_email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}

	return nil
}

// We'll use the Authenticate method to verify whether a user exists with
// the provided email address and password. This will return the relevant
// user ID if they do.
func (m *UserModel) Authenticate(email, password string) (int, error) {
	// Retrieve the id and hashed password associated with the given email. If
	// no matching email exists we return the ErrInvalidCredentials error.
	var id int
	var hashedPassword []byte

	stmt := "SELECT id, hashed_password FROM users WHERE email = ?"

	err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	// Check whether the hashed password and plain-text password provided match.
	// If they don't, we return the ErrInvalidCredentials error.
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	// Otherwise, the password is correct. Return the user ID.
	return id, nil
}

// We'll use the Exists method to check if a user exists with a specific ID.
func (m *UserModel) Exists(id int) (bool, error) {
	var exists bool

	stmt := "SELECT EXISTS(SELECT true FROM users WHERE id = ?)" //使用 SELECT EXISTS 子查询来检查是否存在符合给定 id 的用户

	err := m.DB.QueryRow(stmt, id).Scan(&exists) //执行 SQL 查询语句，并将结果扫描到 exists 变量中
	return exists, err
}

func (m *UserModel) Get(id int) (*User, error) {
	var user User

	stmt := "SELECT id, name, email, created FROM users WHERE id = ?"

	err := m.DB.QueryRow(stmt, id).Scan(&user.ID, &user.Name, &user.Email, &user.Created) //
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		} else {
			return nil, err
		}
	}
	return &user, nil
}

func (m *UserModel) PasswordUpdate(id int, currentPassword, newPassword string) error {
	var currentHashedPassword []byte

	// 根据ID从数据库检索
	stmt := `SELECT hashed_password FROM users WHERE id = ?`

	err := m.DB.QueryRow(stmt, id).Scan(&currentHashedPassword)
	if err != nil {
		return err
	}

	// 当前密码哈希值比较
	err = bcrypt.CompareHashAndPassword(currentHashedPassword, []byte(currentPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return err
	}
	// 更新数据库密码哈希值
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}

	// 插入数据库
	stmt = `UPDATE users SET hashed_password = ? WHERE id = ?`

	_, err = m.DB.Exec(stmt, string(newHashedPassword), id)

	return err
}
