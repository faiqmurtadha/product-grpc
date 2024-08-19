package services

import (
	"context"
	pb "product-grpc/protos/compiled"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductService struct {
	pb.UnimplementedProductServiceServer
	DB *mongo.Collection
}

// Convert MongoDB ObjectID to String
func ObjectIDToString(id primitive.ObjectID) string {
	return id.Hex()
}

// Convert String to MongoDB ObjectID
func StringToObjectID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

func (p *ProductService) GetProducts(c context.Context, pageParam *pb.Filter) (*pb.Products, error) {
	var page int64 = 1
	var limit int64 = 10
	var name string = ""

	if pageParam.GetPage() != 0 {
		page = pageParam.GetPage()
	}
	if pageParam.GetLimit() != 0 {
		limit = pageParam.GetLimit()
	}
	if pageParam.GetName() != "" {
		name = pageParam.GetName()
	}

	var products []*pb.Product
	var pagination = &pb.Pagination{}

	ctx := context.Background()

	skip := (page - 1) * limit

	filter := bson.M{}
	if name != "" {
		filter["name"] = bson.M{"$regex": primitive.Regex{Pattern: ".*" + strings.TrimSpace(name) + ".*", Options: "i"}}
	}

	// Get total data product
	totalCount, err := p.DB.CountDocuments(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Get products
	opts := options.Find().SetSkip(skip).SetLimit(limit)
	rows, err := p.DB.Find(ctx, filter, opts)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer rows.Close(ctx)

	var rawProducts []struct {
		ID    primitive.ObjectID `bson:"_id"`
		Name  string             `bson:"name"`
		Stock uint32             `bson:"stock"`
		Price float64            `bson:"price"`
	}

	if err = rows.All(ctx, &rawProducts); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, raw := range rawProducts {
		products = append(products, &pb.Product{
			Id:    ObjectIDToString(raw.ID),
			Name:  raw.Name,
			Stock: raw.Stock,
			Price: raw.Price,
		})
	}

	pagination.TotalCount = uint64(totalCount)
	pagination.Limit = uint64(limit)
	pagination.Page = uint64(page)

	result := &pb.Products{
		Pagination: pagination,
		Data:       products,
	}

	return result, nil
}

func (p *ProductService) GetProduct(c context.Context, id *pb.Id) (*pb.Product, error) {
	objId, err := StringToObjectID(id.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var rawProduct struct {
		ID    primitive.ObjectID `bson:"_id"`
		Name  string             `bson:"name"`
		Stock uint32             `bson:"stock"`
		Price float64            `bson:"price"`
	}

	if err := p.DB.FindOne(context.TODO(), bson.M{"_id": objId}).Decode(&rawProduct); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "Product not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	product := &pb.Product{
		Id:    ObjectIDToString(rawProduct.ID),
		Name:  rawProduct.Name,
		Stock: rawProduct.Stock,
		Price: rawProduct.Price,
	}

	return product, nil
}

func (p *ProductService) CreateProduct(c context.Context, request *pb.Product) (*pb.Id, error) {
	var objectID primitive.ObjectID
	var err error
	if request.Id == "" {
		objectID = primitive.NewObjectID()
	} else {
		objectID, err = StringToObjectID(request.Id)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	_, err = p.DB.InsertOne(context.TODO(), request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	id := &pb.Id{
		Id: ObjectIDToString(objectID),
	}

	return id, nil
}

func (p *ProductService) UpdateProduct(c context.Context, request *pb.UpdateDataProduct) (*pb.Status, error) {
	objId, err := StringToObjectID(request.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	filter := bson.M{"_id": objId}

	update := bson.M{}
	if request.Name != nil {
		update["name"] = request.Name
	}
	if request.Stock != nil {
		update["stock"] = int32(*request.Stock)
	}
	if request.Price != nil {
		update["price"] = float64(*request.Price)
	}

	if len(update) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No fields to update")
	}

	updateDoc := bson.M{"$set": update}

	_, err = p.DB.UpdateOne(context.TODO(), filter, updateDoc)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	status := &pb.Status{
		Status: "Product Successfully Updated",
	}

	return status, nil
}

func (p *ProductService) DeleteProduct(c context.Context, id *pb.Id) (*pb.Status, error) {
	objId, err := StringToObjectID(id.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	_, err = p.DB.DeleteOne(context.TODO(), bson.M{"_id": objId})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	status := &pb.Status{
		Status: "Product Successfully Deleted",
	}

	return status, nil
}
