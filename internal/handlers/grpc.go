package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jayjaytrn/URLShortener/config"
	"github.com/jayjaytrn/URLShortener/internal/auth"
	"github.com/jayjaytrn/URLShortener/internal/db"
	"github.com/jayjaytrn/URLShortener/internal/db/postgres"
	"github.com/jayjaytrn/URLShortener/internal/types"
	"github.com/jayjaytrn/URLShortener/internal/urlshort"
	"github.com/jayjaytrn/URLShortener/logging"
	pb "github.com/jayjaytrn/URLShortener/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

type URLShortener struct {
	pb.UnimplementedURLShortenerServer
	*grpc.Server
	Storage     db.ShortenerStorage
	Config      *config.Config
	AuthManager *auth.Manager
}

func NewURLShortener(s db.ShortenerStorage, authManager *auth.Manager, cfg *config.Config) *URLShortener {
	return &URLShortener{
		UnimplementedURLShortenerServer: pb.UnimplementedURLShortenerServer{},
		Storage:                         s,
		AuthManager:                     authManager,
		Config:                          cfg,
	}
}

// URLReturner grpc
func (s *URLShortener) URLReturner(ctx context.Context, req *pb.URLReturnerRequest) (*pb.URLReturnerResponse, error) {
	shortURL := req.ShortUrl[len("/"):]
	if shortURL == "" {
		return nil, status.Error(codes.InvalidArgument, "short url is empty")
	}

	originalURL, err := s.Storage.GetOriginal(shortURL)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.URLReturnerResponse{
		OriginalUrl: originalURL,
	}, nil
}

// Shorten grpc
func (s *URLShortener) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	url := req.Url
	valid := urlshort.ValidateURL(url)
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "invalid URL format")
	}

	su, err := urlshort.GenerateShortURL(s.Storage)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	urlData := types.URLData{
		OriginalURL: url,
		ShortURL:    su,
		UserID:      uuid.New().String(),
	}
	err = s.Storage.Put(urlData)
	if err != nil {
		var originalExistErr *postgres.OriginalExistError
		if errors.As(err, &originalExistErr) {
			r := s.Config.BaseURL + "/" + originalExistErr.ShortURL
			shortenResponse := types.ShortenResponse{
				Result: r,
			}

			br, err := json.Marshal(shortenResponse)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			return &pb.ShortenResponse{
				Result: string(br),
			}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	r := s.Config.BaseURL + "/" + su
	shortenResponse := types.ShortenResponse{
		Result: r,
	}
	br, err := json.Marshal(shortenResponse)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ShortenResponse{
		Result: string(br),
	}, nil
}

// ShortenBatch grpc
func (s *URLShortener) ShortenBatch(ctx context.Context, req *pb.ShortenBatchListRequest) (*pb.ShortenBatchListResponse, error) {
	var urls []types.ShortenBatchRequest

	for _, r := range req.Urls {
		urls = append(urls, types.ShortenBatchRequest{
			CorrelationID: r.CorrelationId,
			OriginalURL:   r.OriginalUrl,
		})
	}

	valid := urlshort.ValidateBatchRequestURLs(urls)
	if !valid {
		return nil, status.Error(codes.InvalidArgument, "invalid URL format in batch")
	}

	batchResponse, batchData, err := urlshort.GenerateShortBatch(s.Config, s.Storage, urls, uuid.New().String())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	err = s.Storage.PutBatch(ctx, batchData)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &pb.ShortenBatchListResponse{
		Urls: make([]*pb.ShortenBatchResponse, len(batchResponse)),
	}

	for i, a := range batchResponse {
		response.Urls[i] = &pb.ShortenBatchResponse{
			CorrelationId: a.CorrelationID,
			ShortUrl:      a.ShortURL,
		}
	}

	return response, nil
}

// Urls grpc
func (s *URLShortener) Urls(ctx context.Context, req *pb.UrlsRequest) (*pb.UrlsResponse, error) {
	urls, err := s.Storage.GetURLsByUserID(req.UserId)
	if err != nil {
		if strings.Contains(err.Error(), "no URLs found for userID") {
			return &pb.UrlsResponse{}, nil
		}
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}

	response := &pb.UrlsResponse{
		Urls: make([]*pb.UserURL, len(urls)),
	}

	for i, a := range urls {
		response.Urls[i] = &pb.UserURL{
			ShortUrl:    a.ShortURL,
			OriginalUrl: a.OriginalURL,
		}
	}

	return response, nil
}

// DeleteUrlsAsync grpc
func (s *URLShortener) DeleteUrlsAsync(ctx context.Context, req *pb.DeleteUrlsAsyncRequest) (*pb.DeleteUrlsAsyncResponse, error) {
	shortURLs := req.GetShortUrls()

	if len(shortURLs) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "No URLs provided for deletion")
	}

	// Создаём канал для передачи URL на удаление
	urlChannel := make(chan string)

	// Запуск горутины для отправки URL в канал
	go func() {
		for _, shortURL := range shortURLs {
			urlChannel <- shortURL
		}
		close(urlChannel) // Закрываем канал после передачи всех URL
	}()

	// Запускаем BatchDelete с каналом urlChannel
	go s.Storage.BatchDelete(urlChannel, req.UserId)

	return &pb.DeleteUrlsAsyncResponse{
		Success: true,
	}, nil
}

// Stats grpc
func (s *URLShortener) Stats(ctx context.Context, req *pb.StatsRequest) (*pb.StatsResponse, error) {
	logger := logging.GetSugaredLogger()
	defer logger.Sync()

	trustedSubnet := s.Config.TrustedSubnet
	if trustedSubnet == "" {
		return nil, status.Error(codes.PermissionDenied, "access denied: trusted_subnet is not set")
	}

	// Получаем количество уникальных пользователей и сокращённых URL
	stats, err := s.Storage.GetStats()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.StatsResponse{
		UrlsCount:  int32(stats.Urls),
		UsersCount: int32(stats.Users),
	}, nil
}
