package wallet

import (
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/azizahonohunova/wallet/pkg/types"
	"github.com/google/uuid"
)

var (
	ErrAccountNotFound      = errors.New("Account not found")
	ErrPaymentNotFound      = errors.New("Payment not found")
	ErrAmountMustBePositive = errors.New("amount must be greater > 0")
	ErrPhoneRegistered      = errors.New("phone already registered")
	ErrNotEnoughBalance     = errors.New("not enought balance in account")
	ErrFavoriteNotFound     = errors.New("not find favorite")
	ErrFavoriteAdded        = errors.New("favorite already added")
)

var lastIDofFavorite string

type Service struct {
	nextAccountID int64
	accounts      []*types.Account
	payments      []*types.Payment
	favorites     []*types.Favorite
}

func (s *Service) RegisterAccount(phone types.Phone) (*types.Account, error) {
	for _, account := range s.accounts {
		if account.Phone == phone {
			return nil, ErrPhoneRegistered
		}
	}
	s.nextAccountID++
	account := &types.Account{
		ID:      s.nextAccountID,
		Phone:   phone,
		Balance: 0,
	}
	s.accounts = append(s.accounts, account)
	return account, nil
}
func (s *Service) FindAccountByID(accountID int64) (*types.Account, error) {
	var Account *types.Account
	for _, x := range s.accounts {
		if x.ID == accountID {
			Account = x
		}
	}
	if Account == nil {
		return nil, ErrAccountNotFound
	}
	return Account, nil
}
func (s *Service) FindPaymentByID(paymentID string) (*types.Payment, error) {
	var Payment *types.Payment
	for _, x := range s.payments {
		if x.ID == paymentID {
			Payment = x
		}
	}
	if Payment == nil {
		return nil, ErrPaymentNotFound
	}
	return Payment, nil
}
func (s *Service) Pay(accountID int64, amount types.Money, category types.PaymentCategory) (*types.Payment, error) {
	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}
	account, err := s.FindAccountByID(accountID)
	if err != nil {
		return nil, err
	}
	if account.Balance < amount {
		return nil, ErrNotEnoughBalance
	}
	account.Balance -= amount
	paymentID := uuid.New().String()
	payment := &types.Payment{
		ID:        paymentID,
		AccountID: accountID,
		Amount:    amount,
		Category:  types.PaymentCategory(category),
		Status:    types.PaymentStatusInProgress,
	}
	s.payments = append(s.payments, payment)
	return payment, nil
}
func (s *Service) Reject(paymentID string) error {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return err
	}
	payment.Status = types.PaymentStatusFail
	account, err := s.FindAccountByID(payment.AccountID)
	if err != nil {
		return err
	}
	account.Balance += payment.Amount
	return nil
}
func (s *Service) Deposit(accountID int64, amount types.Money) error {
	if amount <= 0 {
		return ErrAmountMustBePositive
	}
	account, err := s.FindAccountByID(accountID)
	if err != nil {
		return err
	}
	account.Balance += amount
	return nil
}
func (s *Service) Repeat(paymentID string) (*types.Payment, error) {
	newPayment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}
	payment, err := s.Pay(newPayment.AccountID, newPayment.Amount, newPayment.Category)
	if err != nil {
		return nil, err
	}
	return payment, nil
}
func (s *Service) FavoritePayment(paymentID string, name string) (*types.Favorite, error) {
	payment, err := s.FindPaymentByID(paymentID)
	if err != nil {
		return nil, err
	}
	_, err = s.FindFavoriteByID(paymentID)
	if err == nil {
		return nil, ErrFavoriteAdded
	}
	favoriteID := uuid.New().String()
	lastIDofFavorite = favoriteID
	NewFavorite := &types.Favorite{
		ID:        favoriteID,
		AccountID: payment.AccountID,
		Name:      name,
		Amount:    payment.Amount,
		Category:  payment.Category,
	}
	s.favorites = append(s.favorites, NewFavorite)
	return NewFavorite, nil
}
func (s *Service) FindFavoriteByID(favoriteID string) (*types.Favorite, error) {
	for _, x := range s.favorites {
		if x.ID == favoriteID {
			return x, nil
		}
	}
	return nil, ErrFavoriteNotFound
}
func (s *Service) PayFromFavorite(favoriteID string) (*types.Payment, error) {
	favorite, err := s.FindFavoriteByID(favoriteID)
	if err != nil {
		return nil, err
	}
	payment, err := s.Pay(favorite.AccountID, favorite.Amount, favorite.Category)
	if err != nil {
		return nil, err
	}
	return payment, nil
}

// New Functions
func (s *Service) ExportToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	for _, account := range s.accounts {
		err = AddAccountToFile(file, account)
		if err != nil {
			return err
		}
	}

	return nil
}
func Read(file *os.File) ([]byte, error) {
	content := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		read, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		content = append(content, buf[:read]...)
	}
	return content, nil
}
func AddAccountToFile(file *os.File, account *types.Account) error {
	content, err := Read(file)
	if err != nil {
		return err
	}
	str := string(content)

	ID := strconv.Itoa(int(account.ID))
	Phone := string(account.Phone)
	Balance := strconv.Itoa(int(account.Balance))

	str += (ID + ";")
	str += (Phone + ";")
	str += (Balance + "|")
	file.Write([]byte(str))
	return nil
}
func (s *Service) ImportFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	content, err := Read(file)

	var accounts []string = strings.Split(string(content), "|")

	for _, account := range accounts[:len(accounts)-1] {
		var vals []string = strings.Split(account, ";")
		id, err := strconv.Atoi(vals[0])
		if err != nil {
			return err
		}
		balance, err := strconv.Atoi(vals[2])
		if err != nil {
			return err
		}
		newAccount := &types.Account{
			ID:      int64(id),
			Phone:   types.Phone(vals[1]),
			Balance: types.Money(balance),
		}
		s.accounts = append(s.accounts, newAccount)
	}
	return nil
}
