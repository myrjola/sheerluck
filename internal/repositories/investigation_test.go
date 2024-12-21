package repositories_test

import (
	"context"
	"github.com/myrjola/sheerluck/internal/models"
	"github.com/myrjola/sheerluck/internal/repositories"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"testing"
)

func TestInvestigationRepository_Get(t *testing.T) {
	dbs := newTestDB(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := repositories.NewInvestigationRepository(dbs, logger)

	tests := []struct {
		name                  string
		investigationTargetID string
		userID                []byte
		wantInvestigation     models.Investigation
		wantErr               bool
	}{
		{
			name:                  "Without completions",
			investigationTargetID: "rue-morgue",
			userID:                []byte{1},
			wantInvestigation: models.Investigation{
				Target: models.InvestigationTarget{
					ID:        "rue-morgue",
					Name:      "Rue Morgue Murder Scene",
					ShortName: "Rue Morgue",
					Type:      models.InvestigationTargetTypeScene,
					ImagePath: "https://myrjola.twic.pics/sheerluck/rue-morgue.webp",
				},
			},
			wantErr: false,
		},
		{
			name:                  "With completions",
			investigationTargetID: "le-bon",
			userID:                []byte{1},
			wantInvestigation: models.Investigation{
				Target: models.InvestigationTarget{
					ID:        "le-bon",
					Name:      "Adolphe Le Bon",
					ShortName: "Adolphe",
					Type:      models.InvestigationTargetTypePerson,
					ImagePath: "https://myrjola.twic.pics/sheerluck/adolphe_le-bon.webp",
				},
				Completions: []models.Completion{
					{
						ID:       1,
						Order:    0,
						Question: "What is your name?",
						Answer:   "Adolphe Le Bon",
					},
					{
						ID:       2,
						Order:    1,
						Question: "What is your occupation?",
						Answer:   "Bank clerc",
					},
					{
						ID:       3,
						Order:    2,
						Question: "What is your address?",
						Answer:   "Rue Morgue",
					},
				},
			},
			wantErr: false,
		},
		{
			name:                  "Invalid user name returns empty completions",
			investigationTargetID: "rue-morgue",
			userID:                []byte("nonexistent"),
			wantInvestigation: models.Investigation{
				Target: models.InvestigationTarget{
					ID:        "rue-morgue",
					Name:      "Rue Morgue Murder Scene",
					ShortName: "Rue Morgue",
					Type:      models.InvestigationTargetTypeScene,
					ImagePath: "https://myrjola.twic.pics/sheerluck/rue-morgue.webp",
				},
			},
			wantErr: false,
		},
		{
			name:                  "Invalid investigation target",
			investigationTargetID: "nonexistent",
			userID:                []byte{1},
			wantErr:               true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			investigation, err := repo.Get(context.TODO(), tt.investigationTargetID, tt.userID)

			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, investigation, "investigation should be nil")
				return
			}

			require.NoError(t, err, "failed to read investigation")
			require.NotNilf(t, investigation, "investigation not found")
			require.Equal(t, tt.wantInvestigation.Target, investigation.Target, "investigation target mismatch")
			require.Equal(t, tt.wantInvestigation.Completions, investigation.Completions, "completions mismatch")
		})
	}
}
func TestInvestigationRepository_FinishCompletion(t *testing.T) {
	tests := []struct {
		name                  string
		investigationTargetID string
		userID                []byte
		wantCompletion        models.Completion
		wantErr               bool
	}{
		{
			name:                  "First completion",
			investigationTargetID: "rue-morgue",
			userID:                []byte{1},
			wantCompletion: models.Completion{
				Order: 0,
			},
			wantErr: false,
		},
		{
			name:                  "Fourth completion",
			investigationTargetID: "le-bon",
			userID:                []byte{1},
			wantCompletion: models.Completion{
				Order: 3,
			},
			wantErr: false,
		},
		{
			name:                  "invalid investigation target",
			investigationTargetID: "nonexistent",
			userID:                []byte{1},
			wantErr:               true,
		},
		{
			name:                  "invalid user",
			investigationTargetID: "le-bon",
			userID:                []byte("nonexistent"),
			wantErr:               true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dbs := newTestDB(t)
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			repo := repositories.NewInvestigationRepository(dbs, logger)

			err := repo.FinishCompletion(context.TODO(), tt.investigationTargetID, tt.userID, "question", "answer")

			if tt.wantErr {
				require.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "failed to read completion")
			//require.NotNilf(t, completion, "completion not found")
			//require.Equal(t, completion.Order, tt.wantCompletion.Order, "wrong order")
			//require.Equal(t, completion.Question, "question", "question mismatch")
			//require.Equal(t, completion.Answer, "answer", "answer mismatch")
		})
	}
}

func Benchmark_InvestigationRepository(b *testing.B) {
	//readWriteDB, readDB := newBenchmarkDB(b)
	//logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	//repo := NewInvestigationRepository(readWriteDB, readDB, logger)
	//user, err := models.NewUser()
	//require.NoError(b, err)
	//userRepo := NewUserRepository(readWriteDB, readDB, logger)
	//err = userRepo.Upsert(context.Background(), user)
	//require.NoError(b, err)
	//
	//b.ResetTimer()
	//
	//b.RunParallel(func(pb *testing.PB) {
	//	ctx := context.Background()
	//	investigationTarget := "le-bon"
	//	for pb.Next() {
	//		err = repo.FinishCompletion(ctx, investigationTarget, user.ID, "question", "answer")
	//		require.NoError(b, err)
	//		_, err := repo.Get(ctx, investigationTarget, user.ID)
	//		require.NoError(b, err)
	//	}
	//})
}
