package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"database/sql"

	"github.com/google/uuid"
	"github.com/labstack/echo"
	"golang.org/x/crypto/bcrypt"
)

var (
	db  *sql.DB
	err error
)

type user struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type apikey struct {
	API string `json:"api_key"`
}
type todolist struct {
	Item string `json:"item"`
}
type list struct {
	Item      string
	Status    string
	Createdat string
	Updatedat string
}
type todoid struct {
	Taskid int `json:"taskid"`
}
type status struct {
	Taskid int    `json:"taskid"`
	Status string `json:"status"`
}
type newtask struct {
	Taskname string `json:"taskname"`
}

// User Login section API
func userlogin(c echo.Context) error {
	key := uuid.New()
	usr := user{}
	req_body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "Not read your data ")

	}
	err = json.Unmarshal(req_body, &usr)
	password := usr.Password

	// Check whether the password of the user is correct by cross checking the password in the database

	var originalPassword string
	// db, err := sql.Open("mysql", "root:root@(127.0.0.1:3307)/webapp?parseTime=true")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return c.String(http.StatusInternalServerError, "Server is not connected with the database")

	// }

	selectQuery := db.QueryRow("SELECT password FROM user WHERE username = ?", usr.Username).Scan(&originalPassword)
	if selectQuery != nil {
		fmt.Println(selectQuery.Error())
		return c.String(http.StatusInternalServerError, "Database Error")
	}
	compare := bcrypt.CompareHashAndPassword([]byte(originalPassword), []byte(password))

	// If the password which is entered and the original password are equal update the user data in the api table
	secureKey := &apikey{
		API: key.String(),
	}
	if compare != nil {
		return c.String(http.StatusInternalServerError, "User password is incorrect")

	}

	var userid string
	err = db.QueryRow("SELECT  userid FROM user WHERE username=?", usr.Username).Scan(&userid)
	if err != nil {
		fmt.Println(err.Error())
	}

	insertQuery := "INSERT INTO apikeys(userid,apikey,created_date) VALUES (?,?,NOW())"

	resultQuery, err := db.Exec(insertQuery, userid, key.String())

	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "Your data is not saved here")
	}
	fmt.Println(resultQuery.LastInsertId())

	return c.JSON(http.StatusOK, secureKey) // Return the api key for the user

}

// New user registration API

func newuser(c echo.Context) error {
	usr := user{}

	// decoding the json data

	run, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Not read your data ")

	}
	err = json.Unmarshal(run, &usr)
	username := usr.Username

	password := usr.Password

	fmt.Println("username is",username)

	// Connecting with the database

	// db, err := sql.Open("mysql", "root:root@(127.0.0.1:3307)/webapp?parseTime=true")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return c.String(http.StatusInternalServerError, "Server is not connected with the database")

	// }

	// check whether the password is not empty and the username is not duplicate

	var temporary string
	if password == "" {
		return c.String(http.StatusBadRequest, "Empty password not allowed try a new one")

	}
	sqlQuery1, err := db.Query("select username from user")
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "Database error")
	}
	for sqlQuery1.Next() {
		err := sqlQuery1.Scan(&temporary)
		if err != nil {
			fmt.Println(err.Error())
		}
		if username == temporary {
			return c.String(http.StatusInternalServerError, "This name is already taken")

		}
	}

	// Genearate the hashing value for the password and insert the new user to database

	hashValue, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		fmt.Println(err.Error())
	}
	insertQuery := "INSERT INTO user(username,password,created_date) VALUES (?,?,now())"

	resultQuery, err := db.Exec(insertQuery, username, hashValue)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(resultQuery.LastInsertId())

	return c.String(http.StatusOK, "New user is created ")

}

// Create a new todolist with POST request

func todoInsert(c echo.Context) error {
	todo := todolist{}
	reqBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(404, "Invalid content ")

	}

	err = json.Unmarshal(reqBody, &todo)
	userItem := todo.Item
	if userItem == "" {
		return c.String(404, "Invalid content")
	}
	// db, err := sql.Open("mysql", "root:root@(127.0.0.1:3307)/webapp?parseTime=true")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return c.String(http.StatusInternalServerError, "Server is not connected with the database")

	// }
	var userid int
	status := "undone"
	getHeader := c.Request().Header.Get("api-key")

	err = db.QueryRow("SELECT userid FROM apikeys WHERE apikey=?", getHeader).Scan(&userid)
	insertQuery := "INSERT INTO todolist(userid,item,status,created_date,updated_date) VALUES (?,?,?,now(),now())"
	resultQuery, err := db.Exec(insertQuery, userid, todo.Item, status)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "Data not saved in the database")

	}
	defer db.Close()
	resultQuery.LastInsertId()
	return c.String(http.StatusOK, "Success")

}

// Show the current todos with GET request

func presentTodo(c echo.Context) error {
	var userid int
	listslice := []list{} // slice to hold multple rows of a sql query

	getHeader := c.Request().Header.Get("api-key")
	err = db.QueryRow("SELECT userid FROM apikeys WHERE apikey=?", getHeader).Scan(&userid)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "API key is incorrect")
	}
	result, err := db.Query("select item,status,created_date,updated_date from todolist where userid=?", userid)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "Invalid userid")
	}
	for result.Next() {
		listobj := list{}
		err = result.Scan(&listobj.Item, &listobj.Status, &listobj.Createdat, &listobj.Updatedat)
		if err != nil {
			fmt.Println(err.Error())
			return c.String(404, "Not loaded the user data")
		}
		listslice = append(listslice, listobj)
	}

	jsondata, err := json.Marshal(listslice)

	return c.String(http.StatusOK, string(jsondata))

}

// Delete a particular taskid
func deleteTodo(c echo.Context) error {
	taskid := todoid{}
	req_body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "Not read your data ")

	}
	err = json.Unmarshal(req_body, &taskid)
	task := taskid.Taskid
	// db, err := sql.Open("mysql", "root:root@(127.0.0.1:3307)/webapp?parseTime=true")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return c.String(http.StatusInternalServerError, "Server is not connected with the database")

	// }
	deleteQuery := "DELETE FROM todolist WHERE taskid=?"
	execute, err := db.Exec(deleteQuery, task)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(404, "Item not found")

	}

	fmt.Println(execute.LastInsertId())
	defer db.Close()

	return c.String(http.StatusOK, "success")

}

// Update the status of a task
func updateStatus(c echo.Context) error {
	statusObj := status{}
	req_body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(400, "Invalid request")

	}
	err = json.Unmarshal(req_body, &statusObj)
	// db, err := sql.Open("mysql", "root:root@(127.0.0.1:3307)/webapp?parseTime=true")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return c.String(http.StatusInternalServerError, "Server is not connected with the database")

	// }
	updateQuery := "UPDATE todolist SET status=? where taskid=? "
	result, err := db.Exec(updateQuery, statusObj.Status, statusObj.Taskid)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(404, "Item not found")

	}
	defer db.Close()
	fmt.Println(result.LastInsertId())

	return c.String(http.StatusOK, "success")
}
func changeTaskname(c echo.Context) error {
	tasknameobj := newtask{}
	reqBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(404, "Invalid request")
	}
	err = json.Unmarshal(reqBody, &tasknameobj)
	taskid := c.Param("taskid")
	taskname := tasknameobj.Taskname
	db, err := sql.Open("mysql", "root:root@(127.0.0.1:3307)/webapp?parseTime=true")
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "Server is not connected with the database")

	}
	query := "UPDATE  todolist SET item=?,updated_date=now() WHERE taskid=?"
	execute, err := db.Exec(query, taskname,taskid)
	if err != nil {
		fmt.Println(err.Error())
		return c.String(http.StatusInternalServerError, "cannot be updated ")

	}
	fmt.Println(execute.LastInsertId())
	fmt.Println(taskid)

	defer db.Close()

	return c.String(200, "success")
}

// Middleware for todo API

func loginMiddleware(next echo.HandlerFunc) echo.HandlerFunc {

	return func(c echo.Context) error {

		var userid int
		getHeader := c.Request().Header.Get("api-key")

		err := db.QueryRow("SELECT userid FROM apikeys where apikey=?", getHeader).Scan(&userid)
		if err != nil {
			fmt.Println(err.Error())
			return c.String(http.StatusInternalServerError, "api key not found")

		}

		ctx := context.WithValue(c.Request().Context(), "userid", userid)
		newc := c.Request().WithContext(ctx)
		c.SetRequest(newc)
		return next(c)

	}
}

func main() {
	var err error

	db, err = sql.Open("mysql", "root:root@(127.0.0.1:3307)/webapp?parseTime=true")
	if err != nil {
		fmt.Println(err.Error())

	}

	obj := echo.New()
	fmt.Println("started")
	obj.POST("/signup", newuser)
	obj.POST("/signin", userlogin)
	todoMux := obj.Group("/todos", loginMiddleware)

	todoMux.POST("", todoInsert)
	todoMux.GET("", presentTodo)
	todoMux.DELETE("", deleteTodo)
	todoMux.PUT("", updateStatus)
	todoMux.PUT("/:taskid",changeTaskname)

	obj.Start(":8080")

}
