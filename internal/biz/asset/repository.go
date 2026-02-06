// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import "context"

type AssetGroupRepo interface {
	Create(ctx context.Context, group *AssetGroup) error
	Update(ctx context.Context, group *AssetGroup) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*AssetGroup, error)
	GetTree(ctx context.Context) ([]*AssetGroup, error)
	GetAll(ctx context.Context) ([]*AssetGroup, error)
	List(ctx context.Context, page, pageSize int, keyword string) ([]*AssetGroup, int64, error)
	GetDescendantIDs(ctx context.Context, id uint) ([]uint, error)
}

type HostRepo interface {
	Create(ctx context.Context, host *Host) error
	CreateOrUpdate(ctx context.Context, host *Host) error
	Update(ctx context.Context, host *Host) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*Host, error)
	List(ctx context.Context, page, pageSize int, keyword string, groupIDs []uint, accessibleHostIDs []uint, status *int) ([]*Host, int64, error)
	GetByGroupID(ctx context.Context, groupID uint) ([]*Host, error)
	GetByIP(ctx context.Context, ip string) (*Host, error)
	GetByCloudInstanceID(ctx context.Context, instanceID string) (*Host, error)
	CountByCredentialID(ctx context.Context, credentialID uint) (int64, error)
}

type CredentialRepo interface {
	Create(ctx context.Context, credential *Credential) error
	Update(ctx context.Context, credential *Credential) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*Credential, error)
	GetByIDDecrypted(ctx context.Context, id uint) (*Credential, error)
	List(ctx context.Context, page, pageSize int, keyword string) ([]*Credential, int64, error)
	GetAll(ctx context.Context) ([]*Credential, error)
}

type CloudAccountRepo interface {
	Create(ctx context.Context, account *CloudAccount) error
	Update(ctx context.Context, account *CloudAccount) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*CloudAccount, error)
	List(ctx context.Context, page, pageSize int) ([]*CloudAccount, int64, error)
	GetAll(ctx context.Context) ([]*CloudAccount, error)
}
