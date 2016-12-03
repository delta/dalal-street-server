package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/gotest")
	if err != nil {
		fmt.Print(err.Error())
	}

	defer db.Close()
	err = db.Ping()
	if err != nil {
		fmt.Print(err.Error())
	}

	type Session struct {
		Id         int
		Name 			 string  // name
		Emailid		 string  // emailid
		Pragyanid	 string
	}
	router := gin.Default()

	//GET Request for one id
	router.GET("/session/:id", func(c *gin.Context) {
		var (
			session Session
			result gin.H
		)
		id := c.Param("id")
		row := db.QueryRow("select id, name, emailid, pragyanid from session where id = ?;", id)
		err = row.Scan(&session.Id, &session.Name, &session.Emailid, &session.Pragyanid)

		if err != nil {
			result = gin.H{
				"result": "User Not Found",
			}
		} else {
			result = gin.H{
				"result": session,
				"count":  1,
			}
		}
		c.JSON(http.StatusOK, result)
	})
	// GET Request for all the ids
	router.GET("/sessions", func(c *gin.Context) {
		var (
			session  Session
			sessions []Session
		)
		rows, err := db.Query("select id, name, emailid, pragyanid from session;")
		if err != nil {
			fmt.Print(err.Error())
		}

		for rows.Next() {
			err = rows.Scan(&session.Id, &session.Name, &session.Emailid, &session.Pragyanid)
			sessions = append(sessions, session)
			if err != nil {
				fmt.Print(err.Error())
			}
		}

		fmt.Print(sessions)
		defer rows.Close()
		c.JSON(http.StatusOK, gin.H{
			"count":  len(sessions),
			"result": sessions,
		})
	})

// POST Request
	router.POST("/session", func(c *gin.Context) {
		var buffer bytes.Buffer
		name := c.PostForm("name")
		emailid := c.PostForm("emailid")
		pragyanid := c.PostForm("pragyanid")
		stmt, err := db.Prepare("insert into session (name, emailid, pragyanid) values(?,?,?);")

		if err != nil {
			fmt.Print(err.Error())
		}

		_, err = stmt.Exec(name, emailid, pragyanid)

		if err != nil {
			fmt.Print(err.Error())
		}

		buffer.WriteString(name)
		buffer.WriteString(" ")
		buffer.WriteString(emailid)
		buffer.WriteString(" ")
		buffer.WriteString(pragyanid)

		defer stmt.Close()
		Name := buffer.String()
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf(" %s successfully created", Name),
		})
	})

//PUT Request
	router.PUT("/session", func(c *gin.Context) {
		var buffer bytes.Buffer
		id := c.Query("id")
		name := c.PostForm("name")
		emailid := c.PostForm("emailid")
		pragyanid := c.PostForm("pragyanid")

		stmt, err := db.Prepare("update session set name= ?, emailid= ?, pragyanid= ? where id= ?;")

		if err != nil {
			fmt.Print(err.Error())
		}
		_, err = stmt.Exec(name, emailid, pragyanid, id)
		if err != nil {
			fmt.Print(err.Error())
		}

		buffer.WriteString(name)
		buffer.WriteString(" ")
		buffer.WriteString(emailid)
		buffer.WriteString(" ")
		buffer.WriteString(pragyanid)

		defer stmt.Close()
		Name := buffer.String()
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Successfully updated to %s", Name),
		})
	})

//DELETE Request
	router.DELETE("/session", func(c *gin.Context) {

		id := c.Query("id")
		stmt, err := db.Prepare("delete from session where id= ?;")

		if err != nil {
			fmt.Print(err.Error())
		}

		_, err = stmt.Exec(id)
		if err != nil {
			fmt.Print(err.Error())
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("Successfully deleted user: %s", id),
			})
	  }
	})
	router.Run(":3000")
}
