package converter

const accrualMulti = 100

func FormatAccrual(accrual int) float64 {
	return float64(accrual) / accrualMulti
}

func ConvertAccrual(accrual float64) int {
	return int(accrual * accrualMulti)
}
