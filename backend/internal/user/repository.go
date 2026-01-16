package user

import (
	"errors"

	"gorm.io/gorm"
	"temu.in/backend/internal/models"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(u *models.User) error {
	return r.db.Create(u).Error
}

func (r *Repository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindByID(id uint) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) SetRole(id uint, role string) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("role", role).Error
}

func (r *Repository) SavePasswordReset(userID uint, tokenHash string) error {
	return r.db.Exec(`
		INSERT INTO password_resets (user_id, token_hash, expires_at) 
		VALUES (?, ?, NOW() + INTERVAL '1 hour')
	`, userID, tokenHash).Error
}

func (r *Repository) VerifyPasswordResetToken(token string) (uint, error) {
	var result struct {
		UserID uint
	}

	err := r.db.Raw(`
		SELECT user_id FROM password_resets 
		WHERE token_hash = ? AND expires_at > NOW()
	`, token).Scan(&result).Error

	if err != nil {
		return 0, err
	}
	if result.UserID == 0 {
		return 0, errors.New("invalid token")
	}
	return result.UserID, nil
}

func (r *Repository) UpdatePassword(userID uint, hashedPassword string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

func (r *Repository) DeletePasswordResetToken(tokenHash string) error {
	return r.db.Exec("DELETE FROM password_resets WHERE token_hash = ?", tokenHash).Error
}

func (r *Repository) ListAll() ([]models.User, error) {
	var users []models.User
	if err := r.db.Select("id, email, name, role").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *Repository) SaveVerificationToken(userID uint, token string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("verification_token", token).Error
}

func (r *Repository) MarkVerified(userID uint) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_verified":        true,
		"verification_token": nil,
	}).Error
}

func (r *Repository) FindByVerificationToken(token string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("verification_token = ?", token).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
