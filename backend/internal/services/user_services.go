package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/go-ldap/ldap/v3"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserService interface {
	Login(ctx context.Context, req LoginRequest) (*db.ClaimctlUser, error)
	CreateUser(ctx context.Context, req db.CreateUserParams) (*db.ClaimctlUser, error)
	GetUsers(ctx context.Context) (*[]db.ClaimctlUser, error)
	GetUser(ctx context.Context, email string, id uuid.UUID) (*db.ClaimctlUser, error)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	EnsureAdminExists(ctx context.Context) error
	LoginLDAP(ctx context.Context, email, password string) (*db.ClaimctlUser, error)
	LoginOIDC(ctx context.Context, email, name string) (*db.ClaimctlUser, error)
	UpdateChannelConfig(ctx context.Context, userID uuid.UUID, slackDest, teamsUrl, notificationEmail string) (*db.ClaimctlUser, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
}

type UpdateUserRequest struct {
	ID       *uuid.UUID  `json:"id"`
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	Password *string `json:"password"`

	Role   *string `json:"role"`
	Status *string `json:"status"`
}

type userService struct {
	db    db.Querier
	store db.Store
}

func NewUserService(store db.Store) UserService {
	return &userService{db: store, store: store}
}

var dummyHash = []byte("$2a$10$O01o4V/a9qP48WjE5E.eC.Bw3P2B1F4K.iG9k/e1z.w.M1L1T2eGq")

func (s *userService) Login(ctx context.Context, req LoginRequest) (*db.ClaimctlUser, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password must be provided")
	}
	user, err := s.db.FindUserByEmail(ctx, req.Email)
	if err != nil {
		// Prevent timing attack: perform dummy bcrypt check
		bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		return nil, fmt.Errorf("failed to retrieve the user with email %s", req.Email)
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("user %s is inactive", req.Email)
	}

	// Check if account is locked
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now()) {
		return nil, fmt.Errorf("account temporarily locked. Please try again later")
	}

	hashedPassword, err := s.db.GetPasswordById(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrive the password")
	}

	isValidPass := utils.VerifyPassword(req.Password, hashedPassword)
	if !isValidPass {
		lockedUntil := pgtype.Timestamptz{Valid: false}

		var newFailedAttempts int32

		errLock := s.store.ExecTx(ctx, func(q db.Querier) error {
			if user.LockedUntil.Valid && user.LockedUntil.Time.Before(time.Now()) {
				_ = q.ResetUserFailedLoginAttempts(ctx, user.ID)
			}

			// Atomically increment failed attempts
			attempts, errTx := q.UpdateUserFailedLoginAttempts(ctx, db.UpdateUserFailedLoginAttemptsParams{
				LockedUntil: lockedUntil,
				ID:          user.ID,
			})
			if errTx != nil {
				slog.Error("failed to update failed login attempts", "error", errTx)
				newFailedAttempts = user.FailedLoginAttempts + 1
				return errTx
			}
			newFailedAttempts = attempts

			if newFailedAttempts >= 3 {
				lockedUntil = pgtype.Timestamptz{
					Valid: true,
					Time:  time.Now().Add(30 * time.Minute),
				}
				_, _ = q.UpdateUserFailedLoginAttempts(ctx, db.UpdateUserFailedLoginAttemptsParams{
					LockedUntil: lockedUntil,
					ID:          user.ID,
				})
			}
			return nil
		})

		if errLock != nil {
			slog.Error("transaction failed during login attempt processing", "error", errLock)
		}

		if newFailedAttempts >= 3 {
			return nil, fmt.Errorf("account temporarily locked due to too many failed attempts")
		}

		return nil, fmt.Errorf("Invalid password")
	}

	// Reset failed attempts on successful login
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		if user.FailedLoginAttempts > 0 || user.LockedUntil.Valid {
			if resetErr := q.ResetUserFailedLoginAttempts(ctx, user.ID); resetErr != nil {
				slog.Error("failed to reset failed login attempts", "error", resetErr)
				return resetErr
			}
		}

		updateErr := q.UpdateUserLastLogin(ctx, user.ID)
		if updateErr != nil {
			return fmt.Errorf("failed to update last login for user %s", req.Email)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *userService) CreateUser(ctx context.Context, req db.CreateUserParams) (*db.ClaimctlUser, error) {
	if req.Email == "" || req.Name == "" || req.Password == "" {
		return nil, fmt.Errorf("all fields must be provided to create a user")
	}

	if req.Status == "" {
		req.Status = "active"
	}

	if !utils.IsValidEmail(req.Email) {
		return nil, fmt.Errorf("Email must be a valid email address")
	}

	count, err := s.db.VerifyUserEmailIsUnique(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("email %s is already in use", req.Email)
	}

	if count != 0 {
		return nil, fmt.Errorf("email %s is already in use", req.Email)
	}

	if !utils.IsValidPassword(req.Password) {
		return nil, fmt.Errorf(
			"Password must be at least 8 characters long and contain uppercase, lowercase, and special character",
		)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	req.Password = string(hashedPassword)

	err = s.db.CreateUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create new user: %w", err)
	}

	user, err := s.db.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created user: %w", err)
	}
	return &user, nil
}

func (s *userService) GetUsers(ctx context.Context) (*[]db.ClaimctlUser, error) {
	users, err := s.db.FindAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	return &users, nil
}

func (s *userService) GetUser(ctx context.Context, email string, id uuid.UUID) (*db.ClaimctlUser, error) {
	if email != "" {
		user, err := s.db.FindUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		return &user, nil
	} else if id != uuid.Nil {
		user, err := s.db.FindUserById(ctx, id)
		if err != nil {
			return nil, err
		}
		return &user, nil
	}
	return nil, fmt.Errorf("either email or id must be provided to find a user")
}

func (s *userService) UpdateUser(ctx context.Context, req *UpdateUserRequest) error {
	oldUser, err := s.db.FindUserById(ctx, *req.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user with ID %s: %w", *req.ID, err)
	}

	if req.Name == nil || len(*req.Name) == 0 {
		req.Name = &oldUser.Name
	}
	if req.Email == nil || len(*req.Email) == 0 {
		req.Email = &oldUser.Email
	}
	if req.Status == nil || len(*req.Status) == 0 {
		req.Status = &oldUser.Status
	}

	if req.Role == nil || len(*req.Role) == 0 {
		req.Role = &oldUser.Role
	} else if *req.Role != "admin" && *req.Role != "user" {
		return fmt.Errorf("invalid role")
	}

	if req.Password == nil || len(*req.Password) == 0 {
		req.Password = &oldUser.Password
	} else {
		if !utils.IsValidPassword(*req.Password) {
			return fmt.Errorf(
				"Password must be at least 8 characters long and contain uppercase, lowercase, and special character",
			)
		}
		pass, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		*req.Password = string(pass)
	}

	err = s.db.UpdateUserById(ctx, db.UpdateUserByIdParams{
		ID:       *req.ID,
		Email:    *req.Email,
		Name:     *req.Name,
		Password: *req.Password,

		Role:      *req.Role,
		LastLogin: oldUser.LastLogin,
		Status:    *req.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update user with ID %s: %w", req.ID, err)
	}
	return nil
}

func (s *userService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	err := s.db.DeleteUserById(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user with ID %s: %w", id, err)
	}
	return nil
}

func (s *userService) EnsureAdminExists(ctx context.Context) error {
	count, err := s.db.CountAdminUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to count admin users: %w", err)
	}

	if count == 0 {
		slog.Info("No admin users found, creating default admin user")
		return s.createDefaultAdmin(ctx)
	}

	slog.Info("Found existing admin users, skipping default admin creation", "count", count)
	return nil
}

func (s *userService) createDefaultAdmin(ctx context.Context) error {
	adminEmail := utils.GetEnv("ADMIN_EMAIL", "admin@claimctl.com")
	adminPassword := utils.GetEnv("ADMIN_PASSWORD", "adminpassword")
	adminName := utils.GetEnv("ADMIN_USER", "System Administrator")

	// Check if user with this email already exists
	existingCount, err := s.db.VerifyUserEmailIsUnique(ctx, adminEmail)
	if err != nil {
		return fmt.Errorf("failed to verify admin email uniqueness: %w", err)
	}
	if existingCount > 0 {
		slog.Info("User with email already exists, skipping admin creation", "email", adminEmail)
		return nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Create admin user

	role := "admin"
	status := "active"

	err = s.db.CreateUser(ctx, db.CreateUserParams{
		Email:    adminEmail,
		Name:     adminName,
		Password: string(hashedPassword),

		Role:      role,
		Status:    status,
		LastLogin: pgtype.Timestamptz{Valid: false},
	})

	if err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}

	slog.Info("Successfully created default admin user", "email", adminEmail)
	return nil
}

func (s *userService) LoginLDAP(ctx context.Context, email, password string) (*db.ClaimctlUser, error) {
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	l, err := s.connectLDAP()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer l.Close()

	ldapBaseDN := utils.GetEnv("LDAP_BASE_DN", "dc=example,dc=com")
	ldapUserFilter := utils.GetEnv("LDAP_USER_FILTER", "(&(objectClass=person)(uid=%s))")
	searchFilter := fmt.Sprintf(ldapUserFilter, ldap.EscapeFilter(email))

	searchRequest := ldap.NewSearchRequest(
		ldapBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		searchFilter,
		[]string{"dn", "cn", "mail", "memberOf"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP user search failed: %w", err)
	}

	if len(sr.Entries) != 1 {
		return nil, fmt.Errorf("user not found or too many results")
	}

	userEntry := sr.Entries[0]
	userDN := userEntry.DN

	err = l.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("LDAP authentication failed: %w", err)
	}

	return s.syncLDAPUser(ctx, userEntry)
}

func (s *userService) connectLDAP() (*ldap.Conn, error) {
	ldapURL := utils.GetEnv("LDAP_URL", "ldap://localhost:389")
	ldapBindDN := utils.GetEnv("LDAP_BIND_DN", "cn=admin,dc=example,dc=com")
	ldapBindPassword := utils.GetEnv("LDAP_BIND_PASSWORD", "admin")

	l, err := ldap.DialURL(ldapURL)
	if err != nil {
		return nil, err
	}

	err = l.Bind(ldapBindDN, ldapBindPassword)
	if err != nil {
		l.Close()
		return nil, err
	}

	return l, nil
}

func (s *userService) syncLDAPUser(ctx context.Context, entry *ldap.Entry) (*db.ClaimctlUser, error) {
	name := entry.GetAttributeValue("cn")
	email := entry.GetAttributeValue("mail")

	if email == "" {
		return nil, fmt.Errorf("LDAP entry is missing 'mail' attribute")
	}
	if name == "" {
		name = email
	}

	ldapAdminGroupDN := utils.GetEnv("LDAP_ADMIN_GROUP_DN", "cn=admins,dc=example,dc=com")

	role := "user"

	memberOf := entry.GetAttributeValues("memberOf")
	for _, group := range memberOf {
		if strings.EqualFold(group, ldapAdminGroupDN) {

			role = "admin"
			break
		}
	}

	user, err := s.db.FindUserByEmail(ctx, email)
	if err != nil {
		dummyPass := "LDAP_AUTH_" + utils.GenerateRandomString(16)
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(dummyPass), bcrypt.DefaultCost)

		createParams := db.CreateUserParams{
			Email:    email,
			Name:     name,
			Password: string(hashedPassword),

			Role:      role,
			Status:    "active",
			LastLogin: pgtype.Timestamptz{Valid: true, Time: pgtype.Timestamptz{}.Time},
		}

		err = s.db.CreateUser(ctx, createParams)
		if err != nil {
			return nil, fmt.Errorf("failed to sync (create) LDAP user locally: %w", err)
		}

		user, err = s.db.FindUserByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve synced user: %w", err)
		}
	} else {
		err = s.db.UpdateUserById(ctx, db.UpdateUserByIdParams{
			ID:       user.ID,
			Email:    email,
			Name:     name,
			Password: user.Password,

			Role:      role,
			LastLogin: pgtype.Timestamptz{Valid: true, Time: pgtype.Timestamptz{}.Time},
			Status:    user.Status,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to sync (update) LDAP user locally: %w", err)
		}

		user.Role = role
		user.Name = name
	}

	return &user, nil
}

func (s *userService) LoginOIDC(ctx context.Context, email, name string) (*db.ClaimctlUser, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required for OIDC login")
	}
	if name == "" {
		name = email
	}

	user, err := s.db.FindUserByEmail(ctx, email)
	if err != nil {
		dummyPass := "OIDC_AUTH_" + utils.GenerateRandomString(16)
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(dummyPass), bcrypt.DefaultCost)

		createParams := db.CreateUserParams{
			Email:    email,
			Name:     name,
			Password: string(hashedPassword),

			Role:      "user", // Default role
			Status:    "active",
			LastLogin: pgtype.Timestamptz{Valid: true, Time: pgtype.Timestamptz{}.Time},
		}

		err = s.db.CreateUser(ctx, createParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create OIDC user: %w", err)
		}

		user, err = s.db.FindUserByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve created OIDC user: %w", err)
		}
	} else {
		err = s.db.UpdateUserById(ctx, db.UpdateUserByIdParams{
			ID:       user.ID,
			Email:    email,
			Name:     name,
			Password: user.Password,

			Role:      user.Role,
			LastLogin: pgtype.Timestamptz{Valid: true, Time: pgtype.Timestamptz{}.Time},
			Status:    user.Status,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update OIDC user: %w", err)
		}
		user.Name = name
	}

	return &user, nil
}

func (s *userService) UpdateChannelConfig(ctx context.Context, userID uuid.UUID, slackDest, teamsUrl, notificationEmail string) (*db.ClaimctlUser, error) {
	updatedUser, err := s.db.UpdateUserChannelConfig(ctx, db.UpdateUserChannelConfigParams{
		ID:               userID,
		SlackDestination: pgtype.Text{String: slackDest, Valid: slackDest != ""},
		TeamsWebhookUrl:  pgtype.Text{String: teamsUrl, Valid: teamsUrl != ""},
		NotificationEmail: pgtype.Text{String: notificationEmail, Valid: notificationEmail != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user channel config: %w", err)
	}
	return &updatedUser, nil
}

func (s *userService) UpdatePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.db.FindUserById(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	hashedPassword, err := s.db.GetPasswordById(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to retrieve password")
	}

	if !utils.VerifyPassword(currentPassword, hashedPassword) {
		return fmt.Errorf("incorrect current password")
	}

	// Validation for new password
	if !utils.IsValidPassword(newPassword) {
		return fmt.Errorf(
			"New password must be at least 8 characters long and contain uppercase, lowercase, and special character",
		)
	}

	// Hash new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in DB
	err = s.db.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		Password: string(newHashedPassword),
		ID:       userID,
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
