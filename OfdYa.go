package OfdYa

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type ofdYa struct {
	Token string
}

type Receipt struct {
	ID       int
	FP       string
	FD       string
	Date     string
	Products []Product
	Link     string
	Price    int
	VatPrice int
}

type Product struct {
	Name       string
	Quantity   int
	Price      int
	Vat        int
	VatPrice   int
	TotalPrice int
	FP         string
	FD         string
	FN         string
	Time       string
}

type KKT struct {
	Address  string `json:"address"`
	Kktregid string `json:"kktregid"`
}

type KktsOne struct {
	KKT   map[string][]kkt `json:"KKT"`
	Count int              `json:"count"`
}

type kkt struct {
	Address      string `json:"address"`
	Last         string `json:"last"`
	Kktregid     string `json:"kktregid"`
	Turnover     int    `json:"turnover"`
	ReceiptCount int    `json:"receiptCount"`
}

type Documents struct {
	Count     int        `json:"count"`
	Documents []Document `json:"items"`
}

type Document struct {
	DateTime                int               `json:"dateTime"`
	ProvisionSum            int               `json:"provisionSum"`
	FiscalDocumentFormatVer int               `json:"fiscalDocumentFormatVer"`
	Code                    int               `json:"code"`
	FiscalDriveNumber       string            `json:"fiscalDriveNumber"` //ФН
	ShiftNumber             int               `json:"shiftNumber"`
	ReceivingDate           int               `json:"receivingDate"`
	Operator                string            `json:"operator"`
	RequestNumber           int               `json:"requestNumber"`
	EcashTotalSum           int               `json:"ecashTotalSum"`
	FiscalDocumentNumber    int               `json:"fiscalDocumentNumber"` //ФД
	TaxationType            int               `json:"taxationType"`
	NdsNo                   int               `json:"ndsNo"`
	Nds0                    int               `json:"nds0"`
	Nds10                   int               `json:"nds10"`
	Nds18                   int               `json:"nds18"`
	Nds20                   int               `json:"nds20"`
	UserInn                 string            `json:"userInn"`
	CreditSum               int               `json:"creditSum"`
	KktRegId                string            `json:"kktRegId"`
	CashTotalSum            int               `json:"cashTotalSum"`
	TotalSum                int               `json:"totalSum"`
	AuthorityUri            string            `json:"authorityUri"`
	RetailAddress           string            `json:"retailAddress"`
	FiscalSign              int               `json:"fiscalSign"` //ФП
	OperationType           int               `json:"operationType"`
	PrepaidSum              int               `json:"prepaidSum"`
	RetailPlace             string            `json:"retailPlace"`
	User                    string            `json:"user"`
	Products                []ProductDocument `json:"items"`
	Link                    string
	Ofd                     string
}

type ProductDocument struct {
	Quantity    json.Number `json:"quantity"`
	Price       int         `json:"price"`
	Name        string      `json:"name"`
	Sum         int         `json:"sum"`
	ProductType int         `json:"productType"`
	PaymentType int         `json:"paymentType"`
}

type Link struct {
	Link string `json:"link"`
}

func OfdYa(apiKey string) *ofdYa {
	return &ofdYa{
		Token: apiKey,
	}
}

func (ofd *ofdYa) GetReceipts(date time.Time) (receipts []Receipt, err error) {
	kkts, err := ofd.getKKT(date)
	for _, kkt := range kkts {
		r, err := ofd.getDocuments(kkt.Kktregid, date)
		if err != nil {
			return receipts, err
		}
		receipts = append(receipts, r...)
	}
	return receipts, err
}

func (ofd *ofdYa) getKKT(date time.Time) (kkt []KKT, err error) {
	k := &KktsOne{}
	_, err = resty.New().R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Ofdapitoken", ofd.Token).
		SetBody(map[string]interface{}{"date": date.Format("2001-01-01")}).
		SetResult(k).
		Post("https://api.ofd-ya.ru/ofdapi/v2/KKT")

	if err != nil {
		log.Printf("[OFDYA] GetKKT: %s", err.Error())
	}

	for index, value := range k.KKT {
		if k.Count > 0 {
			for _, v := range value {
				kkt = append(kkt, KKT{
					Address:  v.Address,
					Kktregid: index,
				})
			}
		}
	}
	return kkt, err
}

func (ofd *ofdYa) getDocuments(kkt string, date time.Time) (documents []Receipt, err error) {
	docs := &Documents{}
	_, err = resty.New().R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Ofdapitoken", ofd.Token).
		SetBody(map[string]interface{}{"date": date.Format("2006-01-02"), "fiscalDriveNumber": kkt}).
		SetResult(docs).
		Post("https://api.ofd-ya.ru/ofdapi/v1/documents")

	if err != nil {
		log.Printf("[OFDYA] GetDocuments: %s", err.Error())
	}

	for _, document := range docs.Documents {
		link, err := ofd.getLink(kkt, document.FiscalDocumentNumber)
		if err != nil {
			log.Printf("[OFDYA] getLink: %s", err.Error())
		}
		var products []Product
		for _, pr := range document.Products {
			q, _ := strconv.Atoi(pr.Quantity.String())
			products = append(products, Product{
				Name:       pr.Name,
				Quantity:   q,
				Price:      pr.Price,
				Vat:        0,
				VatPrice:   0,
				TotalPrice: pr.Sum,
				FP:         strconv.Itoa(document.FiscalSign),
				FD:         strconv.Itoa(document.FiscalDocumentNumber),
				FN:         document.FiscalDriveNumber,
				Time:       time.Unix(int64(document.DateTime), 0).Format(time.RFC3339),
			})
		}
		documents = append(documents, Receipt{
			ID:       0,
			FP:       strconv.Itoa(document.FiscalSign),
			FD:       strconv.Itoa(document.FiscalDocumentNumber),
			Date:     time.Unix(int64(document.DateTime), 0).Format(time.RFC3339),
			Products: products,
			Link:     link,
			Price:    document.TotalSum,
			VatPrice: 0,
		})
	}

	return documents, err
}

func (ofd *ofdYa) getLink(kkt string, fdn int) (link string, err error) {
	l := &Link{}
	_, err = resty.New().R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Ofdapitoken", ofd.Token).
		SetBody(map[string]interface{}{"fiscalDriveNumber": kkt, "fiscalDocumentNumber": fdn}).
		SetResult(l).
		Post("https://api.ofd-ya.ru/ofdapi/v1/getChequeLink")
	return l.Link, err
}
