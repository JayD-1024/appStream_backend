package service

import (
    "mime/multipart"

	"appstore/backend"
	"appstore/constants"
	"appstore/gateway/stripe"
	"appstore/model"
	"errors"
	"fmt"
	"reflect"

	"github.com/olivere/elastic/v7"
)

func CheckUser(username, password string) (bool, error) {
    query := elastic.NewBoolQuery()
    query.Must(elastic.NewTermQuery("username", username))
    query.Must(elastic.NewTermQuery("password", password))
    searchResult, err := backend.ESBackend.ReadFromES(query, constants.USER_INDEX)
    if err != nil {
        return false, err
    }

    var utype model.User
    for _, item := range searchResult.Each(reflect.TypeOf(utype)) {
        u := item.(model.User)
        if u.Password == password {
            fmt.Printf("Login as %s\n", username)
            return true, nil
        }
    }
    return false, nil
}

func AddUser(user *model.User) (bool, error) {
    // ES check user exist
    query := elastic.NewTermQuery("username", user.Username)
    searchResult, err := backend.ESBackend.ReadFromES(query, constants.USER_INDEX)
    if err != nil {
        return false, err
    }

    if searchResult.TotalHits() > 0 {
        return false, nil
    }

    // Save to ES
    err = backend.ESBackend.SaveToES(user, constants.USER_INDEX, user.Username)
    if err != nil {
        return false, err
    }
    fmt.Printf("User is added: %s\n", user.Username)
    return true, nil
}

func SaveApp(app *model.App, file multipart.File) error {
    // Create productID, priceID
    productID, priceID, err := stripe.CreateProductWithPrice(app.Title, app.Description, int64(app.Price*100))
    if err != nil {
        fmt.Printf("Failed to create Product and Price using Stripe SDK %v\n", err)
        return err
    }
    app.ProductID = productID
    app.PriceID = priceID
    
    // Save to GCS
    medialink, err := backend.GCSBackend.SaveToGCS(file, app.Id)
    if err != nil {
        return err
    }
    app.Url = medialink

    // Save to ES
    err = backend.ESBackend.SaveToES(app, constants.APP_INDEX, app.Id)
    if err != nil {
        fmt.Printf("Failed to save app to elastic search with app index %v\n", err)
        return err
    }
    fmt.Println("App is saved successfully to ES app index.")
 
    return nil
 
 }
 
func SearchApps(title string, description string) ([]model.App, error) {
   if title == "" {
       return SearchAppsByDescription(description)
   }
   if description == "" {
       return SearchAppsByTitle(title)
   }


   query1 := elastic.NewMatchQuery("title", title)
   query1.Operator("AND")
   query2 := elastic.NewMatchQuery("description", description)
   query2.Operator("AND")
   query := elastic.NewBoolQuery().Must(query1, query2)
   searchResult, err := backend.ESBackend.ReadFromES(query, constants.APP_INDEX)
   if err != nil {
       return nil, err
   }


   return getAppFromSearchResult(searchResult), nil
}


func SearchAppsByTitle(title string) ([]model.App, error) {
   query := elastic.NewMatchQuery("title", title)
   query.Operator("AND")
   if title == "" {
       query.ZeroTermsQuery("all")
   }
   searchResult, err := backend.ESBackend.ReadFromES(query, constants.APP_INDEX)
   if err != nil {
       return nil, err
   }

   return getAppFromSearchResult(searchResult), nil
}


func SearchAppsByDescription(description string) ([]model.App, error) {
   query := elastic.NewMatchQuery("description", description)
   query.Operator("AND")
   if description == "" {
       query.ZeroTermsQuery("all")
   }
   searchResult, err := backend.ESBackend.ReadFromES(query, constants.APP_INDEX)
   if err != nil {
       return nil, err
   }


   return getAppFromSearchResult(searchResult), nil
}



func getAppFromSearchResult(searchResult *elastic.SearchResult) []model.App {
   var ptype model.App
   var apps []model.App
   for _, item := range searchResult.Each(reflect.TypeOf(ptype)) {
       p := item.(model.App)
       apps = append(apps, p)
   }
   return apps
}

func SearchAppByID(appID string) (*model.App, error) {
    query := elastic.NewTermQuery("id", appID)
    searchResult, err := backend.ESBackend.ReadFromES(query, constants.APP_INDEX)
    if err != nil {
        return nil, err
    }
    results := getAppFromSearchResult(searchResult)
    if len(results) == 1 {
        return &results[0], nil
    }
    return nil, nil
 }
 
 func CheckoutApp(domain string, appID string) (string, error) {
    app, err := SearchAppByID(appID)
    if err != nil {
        return "", err
    }
    if app == nil {
        return "", errors.New("unable to find app in elasticsearch")
    }
    return stripe.CreateCheckoutSession(domain, app.PriceID)
 }
 