package api

import (
	"cf/net"
	"cf"
)

type FakeUserRepository struct {
	CreateUserUser cf.User
	CreateUserExists bool

	FindByUsernameUsername string
	FindByUsernameUser cf.User
	FindByUsernameNotFound bool

	DeleteUser cf.User

	SetOrgRoleUser cf.User
	SetOrgRoleOrganization cf.Organization
	SetOrgRoleRole string
}

func (repo *FakeUserRepository) FindByUsername(username string) (user cf.User, apiResponse net.ApiResponse) {
	repo.FindByUsernameUsername = username
	user = repo.FindByUsernameUser

	if repo.FindByUsernameNotFound {
		apiResponse = net.NewNotFoundApiResponse("User not found")
	}

	return
}

func (repo *FakeUserRepository) Create(user cf.User) (apiResponse net.ApiResponse) {
	repo.CreateUserUser = user

	if repo.CreateUserExists {
		apiResponse = net.NewApiResponse("User already exists", cf.USER_EXISTS, 400)
	}

	return
}

func (repo *FakeUserRepository) Delete(user cf.User) (apiResponse net.ApiResponse) {
	repo.DeleteUser = user
	return
}

func (repo *FakeUserRepository) SetOrgRole(user cf.User, org cf.Organization, role string) (apiResponse net.ApiResponse) {
	repo.SetOrgRoleUser = user
	repo.SetOrgRoleOrganization = org
	repo.SetOrgRoleRole = role
	return
}