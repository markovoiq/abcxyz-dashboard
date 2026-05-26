package main

import (
	"encoding/csv"
	"html/template"
	"log"
	"math"
	"net/http"
	//"os"
	"sort"
	"strconv"
)

type Sale struct {
	Date     string
	Product  string
	Price    float64
	Quantity int
}

type Product struct {
	Name    string
	Revenue float64
	ABC     string
	XYZ     string
}

type Matrix struct {
    AX []string
    AY []string
    AZ []string
    BX []string
    BY []string
    BZ []string
    CX []string
    CY []string
    CZ []string
}

type DashboardData struct {
	Products        []Product
	TotalRevenue    float64
	TotalProducts   int
	CountA          int
	CountAX         int
	ShareA          float64
	Matrix          Matrix
}

func main() {
	http.HandleFunc("/", uploadPage)
	http.HandleFunc("/upload", uploadHandler)

	log.Println("Сервер запущен на http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func uploadPage(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("templates/index.html")
	tmpl.Execute(w, nil)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Ошибка загрузки файла", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Println("Ошибка чтения CSV:", err)
		http.Error(w, "Ошибка чтения CSV", 500)
		return
	}

	var sales []Sale

	for i, row := range records {
		if i == 0 {
			continue // пропускаем заголовок
		}

		if len(row) < 4 {
			continue
		}

		price, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			continue
		}

		quantity, err := strconv.Atoi(row[3])
		if err != nil {
			continue
		}

		sales = append(sales, Sale{
			Date:     row[0],
			Product:  row[1],
			Price:    price,
			Quantity: quantity,
		})
	}

	products := calculateAnalytics(sales)

	matrix := buildMatrix(products)

	total, countA, countAX, shareA := calculateKPI(products)

	data := DashboardData{
		Products:      products,
		TotalRevenue:  total,
		TotalProducts: len(products),
		CountA:        countA,
		CountAX:       countAX,
		ShareA:        shareA,
		Matrix:        matrix,
	}

	tmpl, err := template.New("dashboard.html").Funcs(template.FuncMap{
		"formatNumber": formatNumber,
		"formatPercent": formatPercent,
	}).ParseFiles("templates/dashboard.html")

	if err != nil {
		log.Println(err)
		http.Error(w, "Ошибка шаблона", 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		log.Println(err)
		http.Error(w, "Ошибка отображения", 500)
	}
}

func calculateAnalytics(sales []Sale) []Product {

	revenueMap := make(map[string]float64)
	quantityMap := make(map[string][]float64)

	for _, s := range sales {
		revenue := s.Price * float64(s.Quantity)
		revenueMap[s.Product] += revenue
		quantityMap[s.Product] = append(quantityMap[s.Product], float64(s.Quantity))
	}

	var products []Product
	var totalRevenue float64

	for name, revenue := range revenueMap {
		totalRevenue += revenue
		products = append(products, Product{
			Name:    name,
			Revenue: revenue,
		})
	}

	// сортировка по убыванию выручки
	sort.Slice(products, func(i, j int) bool {
		return products[i].Revenue > products[j].Revenue
	})

	// ===== ABC анализ =====
	var cumulative float64

	for i := range products {
		share := products[i].Revenue / totalRevenue
		cumulative += share

		if cumulative <= 0.8 {
			products[i].ABC = "A"
		} else if cumulative <= 0.95 {
			products[i].ABC = "B"
		} else {
			products[i].ABC = "C"
		}
	}

	// ===== XYZ анализ =====
	for i := range products {

		quantities := quantityMap[products[i].Name]

		mean := mean(quantities)
		stdDev := stdDev(quantities, mean)

		if len(quantities) == 0 {
			products[i].XYZ = "Z"
			continue
		}

		if mean == 0 {
			products[i].XYZ = "Z"
			continue
		}

		cv := stdDev / mean

		if cv <= 0.3 {
			products[i].XYZ = "X"
		} else if cv <= 0.7 {
			products[i].XYZ = "Y"
		} else {
			products[i].XYZ = "Z"
		}
	}

	return products
}

func buildMatrix(products []Product) Matrix {

	var matrix Matrix

	for _, p := range products {
		switch p.ABC + p.XYZ {
		case "AX":
			matrix.AX = append(matrix.AX, p.Name)
		case "AY":
			matrix.AY = append(matrix.AY, p.Name)
		case "AZ":
			matrix.AZ = append(matrix.AZ, p.Name)
		case "BX":
			matrix.BX = append(matrix.BX, p.Name)
		case "BY":
			matrix.BY = append(matrix.BY, p.Name)
		case "BZ":
			matrix.BZ = append(matrix.BZ, p.Name)
		case "CX":
			matrix.CX = append(matrix.CX, p.Name)
		case "CY":
			matrix.CY = append(matrix.CY, p.Name)
		case "CZ":
			matrix.CZ = append(matrix.CZ, p.Name)
		}
	}

	return matrix
}

func calculateKPI(products []Product) (float64, int, int, float64) {

	var total float64
	var countA int
	var countAX int

	for _, p := range products {
		total += p.Revenue

		if p.ABC == "A" {
			countA++
		}

		if p.ABC == "A" && p.XYZ == "X" {
			countAX++
		}
	}

	var shareA float64
	if len(products) > 0 {
		shareA = float64(countA) / float64(len(products)) * 100
	}

	return total, countA, countAX, shareA
}

func mean(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	var sum float64
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return math.Sqrt(sum / float64(len(values)))
}

func formatNumber(n float64) string {
	return strconv.FormatFloat(n, 'f', 0, 64)
}

func formatPercent(n float64) string {
	return strconv.FormatFloat(n, 'f', 1, 64)
}