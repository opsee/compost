package com

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"golang.org/x/crypto/bcrypt"
	"time"
)

const (
	BastionStateNew       = "new"
	BastionStateLaunching = "launching"
	BastionStateFailed    = "failed"
	BastionStateActive    = "active"
	BastionStateDisabled  = "disabled"
	BastionStateDeleted   = "deleted"
)

var BastionStates = []string{
	BastionStateNew,
	BastionStateLaunching,
	BastionStateFailed,
	BastionStateActive,
	BastionStateDisabled,
	BastionStateDeleted,
}

type BastionUser struct {
	Username string
	Key      string
}

type Bastion struct {
	ID            string         `json:"id"`
	CustomerID    string         `json:"customer_id" db:"customer_id"`
	UserID        int            `json:"user_id" db:"user_id"`
	StackID       sql.NullString `json:"stack_id" db:"stack_id"`
	ImageID       sql.NullString `json:"image_id" db:"image_id"`
	InstanceID    sql.NullString `json:"instance_id" db:"instance_id"`
	GroupID       sql.NullString `json:"group_id" db:"group_id"`
	InstanceType  string         `json:"instance_type" db:"instance_type"`
	SubnetRouting string         `json:"subnet_routing" db:"subnet_routing"`
	VPCID         string         `json:"vpc_id" db:"vpc_id"`
	SubnetID      string         `json:"subnet_id" db:"subnet_id"`
	State         string         `json:"state"`
	Connected     bool           `json:"connected"`
	Password      string         `json:"-"`
	PasswordHash  string         `json:"-" db:"password_hash"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

func NewBastion(userID int, customerID, vpcID, subnetID, subnetRouting, instanceType string) (*Bastion, error) {
	pwbytes := make([]byte, 18)
	if _, err := rand.Read(pwbytes); err != nil {
		return nil, err
	}

	pw := base64.StdEncoding.EncodeToString(pwbytes)
	pwhash, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	if err != nil {
		return nil, err
	}

	return &Bastion{
		Password:      pw,
		PasswordHash:  string(pwhash),
		CustomerID:    customerID,
		UserID:        userID,
		InstanceType:  instanceType,
		SubnetRouting: subnetRouting,
		State:         BastionStateNew,
		VPCID:         vpcID,
		SubnetID:      subnetID,
		StackID:       sql.NullString{},
		ImageID:       sql.NullString{},
		InstanceID:    sql.NullString{},
		GroupID:       sql.NullString{},
	}, nil
}

func (bastion *Bastion) StackName() string {
	return "opsee-bastion-" + bastion.ID
}

func (bastion *Bastion) Name() string {
	return "Opsee Bastion " + bastion.ID
}

func (bastion *Bastion) Fail() *Bastion {
	bastion.State = BastionStateFailed
	return bastion
}

func (bastion *Bastion) Launch(stackID, imageID string) *Bastion {
	bastion.State = BastionStateLaunching
	bastion.StackID = sql.NullString{stackID, stackID != ""}
	bastion.ImageID = sql.NullString{imageID, imageID != ""}
	return bastion
}

func (bastion *Bastion) Activate(instanceID, groupID string) *Bastion {
	bastion.State = BastionStateActive
	bastion.InstanceID = sql.NullString{instanceID, instanceID != ""}
	bastion.GroupID = sql.NullString{groupID, groupID != ""}
	return bastion
}

func (bastion *Bastion) Authenticate(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(bastion.PasswordHash), []byte(password))
}
