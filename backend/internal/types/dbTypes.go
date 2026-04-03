package types

type JSONB map[string]any

type JSONBArray []any

type JSONBMap map[string]string

type UpdateUserRequest struct {
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	Password *string `json:"password"`
	Admin    *bool   `json:"admin"` // Pointer allows nil detection
	Role     *string `json:"role"`
	Status   *string `json:"status"`
}
