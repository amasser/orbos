package scripts

func GetAll() map[string]string {
	return map[string]string{
		"V1.0__databases.sql":            V10Databases,
		"V1.1__eventstore.sql":           V11Eventstore,
		"V1.2__views.sql":                V12Views,
		"V1.3__usermembership.sql":       V13Usermembership,
		"V1.4__compliance.sql":           V14Compliance,
		"V1.5__orgdomain_validationtype": V15OrgDomainValidationType,
	}
}
