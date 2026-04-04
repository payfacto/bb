package bitbucket

import "context"

// UserResource provides operations on the authenticated user.
type UserResource struct {
	client *Client
}

// Me returns the authenticated user's profile.
func (r *UserResource) Me(ctx context.Context) (User, error) {
	data, err := r.client.do(ctx, "GET", "/user", nil, nil)
	if err != nil {
		return User{}, err
	}
	return decode[User](data)
}
