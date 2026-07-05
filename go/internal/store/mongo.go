package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Colecciones
const (
	colModelos      = "modelos"
	colPredicciones = "predicciones"
	colUsuarios     = "usuarios"
)

type MongoStore struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewMongoStore conecta a MongoDB, verifica la conexión y asegura las colecciones.
func NewMongoStore(ctx context.Context, uri string) (*MongoStore, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo: ping: %w", err)
	}
	db := client.Database("riesgo_delictivo")
	s := &MongoStore{client: client, db: db}
	if err := s.asegurarColecciones(ctx); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return s, nil
}

// crea si no existen las colecciones
func (s *MongoStore) asegurarColecciones(ctx context.Context) error {
	for _, name := range []string{colModelos, colPredicciones, colUsuarios} {
		if err := s.db.CreateCollection(ctx, name); err != nil {
			if merr, ok := err.(mongo.CommandError); ok && merr.Name == "NamespaceExists" {
				continue
			}
			return fmt.Errorf("mongo: crear colección %s: %w", name, err)
		}
	}
	return nil
}

// guarda un checkpoint de entrenamiento en "modelos" con pesos, epoch, costo, marca de finalización y timestamp.
func (s *MongoStore) GuardarCheckpoint(ctx context.Context, pesos []float64, epoch int, costo float64, isFinal bool) error {
	doc := bson.M{
		"pesos":      pesos,
		"epoch":      epoch,
		"costo":      costo,
		"is_final":   isFinal,
		"created_at": time.Now().UTC(),
	}
	_, err := s.db.Collection(colModelos).InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("mongo: insertar checkpoint: %w", err)
	}
	return nil
}

// devuelve el ultimo modelo guardado
func (s *MongoStore) CargarUltimoModelo(ctx context.Context) ([]float64, int, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	var doc struct {
		Pesos []float64 `bson:"pesos"`
		Epoch int       `bson:"epoch"`
	}
	if err := s.db.Collection(colModelos).FindOne(ctx, bson.D{}, opts).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, 0, fmt.Errorf("mongo: no hay modelos guardados")
		}
		return nil, 0, fmt.Errorf("mongo: cargar último modelo: %w", err)
	}
	return doc.Pesos, doc.Epoch, nil
}

// RegistrarPrediccion inserta un documento en "predicciones" con los inputs
// (hora, barrio_id, dia_semana), probabilidad, marca desde_cache y timestamp.
func (s *MongoStore) RegistrarPrediccion(ctx context.Context, hora, barrioID, diaSemana int, prob float64, desdeCache bool) error {
	doc := bson.M{
		"hora":         hora,
		"barrio_id":    barrioID,
		"dia_semana":   diaSemana,
		"probabilidad": prob,
		"desde_cache":  desdeCache,
		"timestamp":    time.Now().UTC(),
	}
	if _, err := s.db.Collection(colPredicciones).InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("mongo: registrar predicción: %w", err)
	}
	return nil
}

// Usuario representa un usuario registrado en el sistema.
type Usuario struct {
	Email    string `bson:"email"`
	Password string `bson:"password"` // hash bcrypt
	Rol      string `bson:"rol"`      // "admin" o "usuario"
}

// RegistrarUsuario inserta un nuevo usuario. Si el email ya existe, devuelve error.
func (s *MongoStore) RegistrarUsuario(ctx context.Context, email, passwordHash string) error {
	// Verificar si ya existe
	var existente Usuario
	err := s.db.Collection(colUsuarios).FindOne(ctx, bson.M{"email": email}).Decode(&existente)
	if err == nil {
		return fmt.Errorf("mongo: el email %s ya está registrado", email)
	}
	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("mongo: verificando email: %w", err)
	}

	doc := bson.M{
		"email":    email,
		"password": passwordHash,
		"rol":      "usuario",
	}
	if _, err := s.db.Collection(colUsuarios).InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("mongo: insertar usuario: %w", err)
	}
	return nil
}

// BuscarUsuario busca un usuario por email. Devuelve nil si no existe.
func (s *MongoStore) BuscarUsuario(ctx context.Context, email string) (*Usuario, error) {
	var u Usuario
	err := s.db.Collection(colUsuarios).FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mongo: buscar usuario: %w", err)
	}
	return &u, nil
}

// Close desconecta del cliente MongoDB.
func (s *MongoStore) Close(ctx context.Context) error {
	if s.client == nil {
		return nil
	}
	return s.client.Disconnect(ctx)
}
