package models

import (
	"gorm.io/gorm"
)

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex"`
	Password  string
	IsAdmin   bool
	CreatedAt int64
	Email     string
	MD5Hash   string // for gravatar, if no Email is specified, the value is random

	Gists   []Gist   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:UserID"`
	SSHKeys []SSHKey `gorm:"foreignKey:UserID"`
	Liked   []Gist   `gorm:"many2many:likes;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (user *User) BeforeDelete(tx *gorm.DB) error {
	// Decrement likes counter for all gists liked by this user
	// The likes will be automatically deleted by the foreign key constraint
	err := tx.Model(&Gist{}).
		Omit("updated_at").
		Where("id IN (?)", tx.
			Select("gist_id").
			Table("likes").
			Where("user_id = ?", user.ID),
		).
		UpdateColumn("nb_likes", gorm.Expr("nb_likes - 1")).
		Error
	if err != nil {
		return err
	}

	// Decrement forks counter for all gists forked by this user
	return tx.Model(&Gist{}).
		Omit("updated_at").
		Where("id IN (?)", tx.
			Select("forked_id").
			Table("gists").
			Where("user_id = ?", user.ID),
		).
		UpdateColumn("nb_forks", gorm.Expr("nb_forks - 1")).
		Error
}

func UserExists(username string) (bool, error) {
	var count int64
	err := db.Model(&User{}).Where("username like ?", username).Count(&count).Error
	return count > 0, err
}

func GetAllUsers(offset int) ([]*User, error) {
	var users []*User
	err := db.
		Limit(11).
		Offset(offset * 10).
		Order("id asc").
		Find(&users).Error

	return users, err
}

func GetUserByUsername(username string) (*User, error) {
	user := new(User)
	err := db.
		Where("username like ?", username).
		First(&user).Error
	return user, err
}

func GetUserById(userId uint) (*User, error) {
	user := new(User)
	err := db.
		Where("id = ?", userId).
		First(&user).Error
	return user, err
}

func GetUserBySSHKeyID(sshKeyId uint) (*User, error) {
	user := new(User)
	err := db.
		Preload("SSHKeys").
		Joins("join ssh_keys on users.id = ssh_keys.user_id").
		Where("ssh_keys.id = ?", sshKeyId).
		First(&user).Error

	return user, err
}

func (user *User) Create() error {
	return db.Create(&user).Error
}

func (user *User) Update() error {
	return db.Save(&user).Error
}

func (user *User) Delete() error {
	return db.Delete(&user).Error
}

func (user *User) SetAdmin() error {
	return db.Model(&user).Update("is_admin", true).Error
}

func (user *User) HasLiked(gist *Gist) (bool, error) {
	association := db.Model(&gist).Where("user_id = ?", user.ID).Association("Likes")
	if association.Error != nil {
		return false, association.Error
	}

	if association.Count() == 0 {
		return false, nil
	}
	return true, nil
}

// -- DTO -- //

type UserDTO struct {
	Username string `form:"username" validate:"required,max=24,alphanum,notreserved"`
	Password string `form:"password" validate:"required"`
}

func (dto *UserDTO) ToUser() *User {
	return &User{
		Username: dto.Username,
		Password: dto.Password,
	}
}
