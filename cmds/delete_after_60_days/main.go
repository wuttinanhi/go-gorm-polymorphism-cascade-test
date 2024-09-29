// Testing Go Gorm Polymorphic Relationships
// Child can have multiple parents
// Child must be deleted when parent is deleted
package main

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name  string
	Post  []*Post  `gorm:"foreignKey:UserID"`
	Reply []*Reply `gorm:"foreignKey:UserID"`
}

type Post struct {
	gorm.Model
	Title   string
	Content string
	UserID  uint
	User    User
	Replies []*Reply `gorm:"polymorphic:Parent;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// HardDeleteRepliesAfterNDays hard deletes replies after N days
func HardDeleteRepliesAfterNDays(db *gorm.DB, days int) error {
	return db.Debug().Unscoped().
		Model(&Reply{}).
		Where("deleted_at > ?", time.Now().AddDate(0, 0, -days)).
		Delete(&Reply{}).Error
}

// Add a hook to delete associated replies
func (p *Post) AfterDelete(tx *gorm.DB) error {
	return tx.Where("parent_type = ? AND parent_id = ?", "Post", p.ID).Delete(&Reply{}).Error
}

type Reply struct {
	gorm.Model
	Content    string
	UserID     uint
	User       User
	ParentID   uint
	ParentType string
	Replies    []*Reply    `gorm:"polymorphic:Parent;"`
	Parent     interface{} `gorm:"-"`
}

// prevent reply chain reply1 -> reply2 -> reply3
func (r *Reply) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ParentType == "Reply" {
		// get the parent reply
		var parentReply Reply
		err = tx.First(&parentReply, r.ParentID).Error
		if err != nil {
			return
		}

		// if parent reply has a parent reply, return an error
		if parentReply.ParentType == "Reply" {
			err = fmt.Errorf("Reply can't have a parent reply")
			return
		}
	}
	return
}

func ResetDB(db *gorm.DB) {
	tables, err := db.Migrator().GetTables()
	if err != nil {
		panic(err)
	}
	for _, table := range tables {
		err := db.Migrator().DropTable(table)
		if err != nil {
			panic(err)
		}
	}
}

func PanicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func PrettyPrintReply(replies []*Reply, prefix string) {
	for _, reply := range replies {
		fmt.Printf("%s Reply ID: %d | Content: %s\n", prefix, reply.ID, reply.Content)

		// if len(reply.Replies) > 0 {
		// 	PrettyPrintReply(reply.Replies, "---")
		// }
	}
}

func main() {
	// Initialize your database connection here
	dsn := "host=localhost user=postgres password=postgres dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// reset the database
	ResetDB(db)

	// Migrate the schema
	err = db.AutoMigrate(&User{}, &Post{}, &Reply{})
	PanicIfError(err)

	// create a user
	user := User{
		Name: "User A",
	}
	err = db.Create(&user).Error
	PanicIfError(err)

	// create a post
	post := &Post{
		User:    user,
		Title:   "Post Title",
		Content: "Post Content",
	}
	err = db.Create(post).Error
	PanicIfError(err)

	for i := 0; i < 10; i++ {
		// create a reply1 for the post
		reply1 := &Reply{
			User:       user,
			Content:    fmt.Sprintf("Reply #%d", i),
			ParentID:   post.ID,
			ParentType: "Post",
			Parent:     post,
		}
		err = db.Create(reply1).Error
		PanicIfError(err)

	}

	dayMinus61 := time.Now().AddDate(0, 0, -61)

	// pretty print date
	fmt.Println(dayMinus61.Format("2006/01/02 15:04"))
	fmt.Println("now", time.Now().Format("2006/01/02 15:04"))

	// delete replies
	err = db.Model(&Reply{}).Where("parent_id = ? AND parent_type = ?", post.ID, "Post").Delete(&Reply{}).Error
	PanicIfError(err)

	// update first 5 deleted reply to 60 days ago
	var replies []*Reply
	err = db.Unscoped().Model(&Reply{}).Where("parent_id = ? AND parent_type = ?", post.ID, "Post").Find(&replies).Error
	PanicIfError(err)

	// PrettyPrintReply(replies, "")
	for i, reply := range replies {
		if i < 5 {
			err = db.Unscoped().Model(reply).Update("deleted_at", dayMinus61).Error
			PanicIfError(err)
		}
	}

	// delete the replies after 60 days
	HardDeleteRepliesAfterNDays(db, 60)
}
