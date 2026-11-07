package auth

// Role — роль пользователя. Список фиксирован, настройки через интерфейс в MVP нет.
type Role string

const (
	RoleAdmin       Role = "admin"       // администратор
	RoleManager     Role = "manager"     // руководитель
	RoleEmployee    Role = "employee"    // сотрудник
	RoleAccountant  Role = "accountant"  // бухгалтер
	RoleStorekeeper Role = "storekeeper" // кладовщик
)

var AllRoles = []Role{RoleAdmin, RoleManager, RoleEmployee, RoleAccountant, RoleStorekeeper}

func (r Role) Valid() bool {
	for _, x := range AllRoles {
		if x == r {
			return true
		}
	}
	return false
}

// Permission — право на действие. Проверяется в хендлерах через Require.
type Permission string

const (
	PermDirectoriesRead  Permission = "directories.read"
	PermDirectoriesWrite Permission = "directories.write"
	PermCRMRead          Permission = "crm.read"
	PermCRMWrite         Permission = "crm.write"
	PermTreasuryRead     Permission = "treasury.read"
	PermTreasuryCreate   Permission = "treasury.create"
	PermTreasuryApprove  Permission = "treasury.approve"
	PermWMSRead          Permission = "wms.read"
	PermWMSWrite         Permission = "wms.write"
	PermUsersManage      Permission = "users.manage"
	PermAuditRead        Permission = "audit.read"
)

// Матрица прав. Администратор может всё, остальные — по таблице из docs/architecture.md.
var rolePermissions = map[Role][]Permission{
	RoleManager: {
		PermDirectoriesRead, PermDirectoriesWrite,
		PermCRMRead, PermCRMWrite,
		PermTreasuryRead, PermTreasuryCreate, PermTreasuryApprove,
		PermWMSRead,
	},
	RoleEmployee: {
		PermDirectoriesRead,
		PermCRMRead, PermCRMWrite,
		PermTreasuryRead, PermTreasuryCreate,
	},
	RoleAccountant: {
		PermDirectoriesRead, PermDirectoriesWrite,
		PermTreasuryRead, PermTreasuryCreate,
		PermWMSRead,
	},
	RoleStorekeeper: {
		PermDirectoriesRead, PermDirectoriesWrite,
		PermWMSRead, PermWMSWrite,
	},
}

// Can отвечает, есть ли у роли право.
func (r Role) Can(p Permission) bool {
	if r == RoleAdmin {
		return true
	}
	for _, x := range rolePermissions[r] {
		if x == p {
			return true
		}
	}
	return false
}
