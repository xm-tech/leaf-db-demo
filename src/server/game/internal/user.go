package internal

import (
	"server/msg"
	"time"

	"github.com/name5566/leaf/gate"
	g "github.com/name5566/leaf/go"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/timer"
	"github.com/name5566/leaf/util"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	accIDUsers = make(map[string]*User)
	users      = make(map[int]*User)
)

const (
	userLogin = iota
	userLogout
	userGame
)

type User struct {
	// 每个玩家维护 1 Agent, 省去了从 map 里面查 agent 的消耗
	gate.Agent
	*g.LinearContext
	state       int
	data        *UserData
	saveDBTimer *timer.Timer
}

func (user *User) login(accID string) {
	userData := new(UserData)
	skeleton.Go(func() {
		db := mongoDB.Ref()
		defer mongoDB.UnRef(db)

		// load
		err := db.DB("game").C("users").
			Find(bson.M{"accid": accID}).One(userData)
		if err != nil {
			// unknown error
			if err != mgo.ErrNotFound {
				log.Error("load acc %v data error: %v", accID, err)
				userData = nil
				user.WriteMsg(&msg.S2C_Close{Err: msg.S2C_Close_InnerError})
				user.Close()
				return
			}

			// new
			err := userData.initValue(accID)
			if err != nil {
				log.Error("init acc %v data error: %v", accID, err)
				userData = nil
				user.WriteMsg(&msg.S2C_Close{Err: msg.S2C_Close_InnerError})
				user.Close()
				return
			}
		}
	}, func() {
		// network closed
		if user.state == userLogout {
			user.logout(accID)
			return
		}

		// db error
		user.state = userGame
		if userData == nil {
			return
		}

		// ok
		user.data = userData
		// 此处可以用 redis 等中心库，实现游戏服本身的无状态，便于水平扩展db
		user.UserData().(*AgentInfo).userID = userData.UserID
		user.onLogin()
		user.autoSaveDB()
	})
}

func (user *User) logout(accID string) {
	if user.data != nil {
		user.saveDBTimer.Stop()
		user.onLogout()
		delete(users, user.data.UserID)
	}

	// save
	data := util.DeepClone(user.data)
	user.Go(func() {
		if data != nil {
			db := mongoDB.Ref()
			defer mongoDB.UnRef(db)
			userID := data.(*UserData).UserID
			_, err := db.DB("game").C("users").
				UpsertId(userID, data)
			if err != nil {
				log.Error("save user %v data error: %v", userID, err)
			}
		}
	}, func() {
		delete(accIDUsers, accID)
	})
}

func (user *User) autoSaveDB() {
	const duration = 5 * time.Minute

	// save
	user.saveDBTimer = skeleton.AfterFunc(duration, func() {
		data := util.DeepClone(user.data)
		user.Go(func() {
			db := mongoDB.Ref()
			defer mongoDB.UnRef(db)
			userID := data.(*UserData).UserID
			_, err := db.DB("game").C("users").
				UpsertId(userID, data)
			if err != nil {
				log.Error("save user %v data error: %v", userID, err)
			}
		}, func() {
			user.autoSaveDB()
		})
	})
}

func (user *User) isOffline() bool {
	return user.state == userLogout
}

func (user *User) onLogin() {

}

func (user *User) onLogout() {

}
