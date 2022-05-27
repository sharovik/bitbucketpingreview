package database

import "github.com/sharovik/orm/dto"

var UsersTable = dto.BaseModel{
	TableName: "users",
	Fields: []interface{}{
		dto.ModelField{
			Name:       "slack_uuid",
			Type:       dto.VarcharColumnType,
			Length:     255,
			IsNullable: true,
		},
		dto.ModelField{
			Name:       "bitbucket_uuid",
			Type:       dto.VarcharColumnType,
			Length:     255,
			IsNullable: true,
		},
		dto.ModelField{
			Name:       "username",
			Type:       dto.VarcharColumnType,
			Length:     255,
			IsNullable: true,
		},
		dto.ModelField{
			Name:       "nickname",
			Type:       dto.VarcharColumnType,
			Length:     255,
			IsNullable: true,
		},
	},
	PrimaryKey: dto.ModelField{
		Name:          "id",
		Type:          dto.IntegerColumnType,
		AutoIncrement: true,
		IsPrimaryKey:  true,
	},
}
