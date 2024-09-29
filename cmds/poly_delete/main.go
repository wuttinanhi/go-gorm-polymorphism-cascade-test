// Testing Go Gorm Polymorphic Relationships
// Child can have multiple parents
// Child must be deleted when parent is deleted
package main

import (
	"fmt"

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

// Add a hook to delete associated replies
// func (p *Post) AfterDelete(tx *gorm.DB) error {
// 	return tx.Where("parent_type = ? AND parent_id = ?", "Post", p.ID).Delete(&Reply{}).Error
// }

type Reply struct {
	gorm.Model
	Content    string
	UserID     uint
	User       User
	ParentID   uint
	ParentType string
	Replies    []*Reply    `gorm:"polymorphic:Parent;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Parent     interface{} `gorm:"-"`
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

	// create a reply1 for the post
	reply1 := &Reply{
		User:       user,
		Content:    "Reply 1",
		ParentID:   post.ID,
		ParentType: "Post",
		Parent:     post,
	}
	err = db.Create(reply1).Error
	PanicIfError(err)

	// delete the post
	err = db.Delete(post).Error
	PanicIfError(err)

	// count replies after deleting the post
	var count int64
	err = db.Model(&Reply{}).Count(&count).Error
	PanicIfError(err)
	fmt.Println("Replies count after deleting the post:", count)
}
