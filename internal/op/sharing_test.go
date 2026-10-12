package op_test

import (
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

func TestGetSharingUnwrapPath(t *testing.T) {
	adminUser := &model.User{
		Role:     model.ADMIN,
		BasePath: "/",
	}

	tests := []struct {
		name        string
		sharing     *model.Sharing
		path        string
		want        string
		expectError bool
	}{
		{
			name: "single file - root path",
			sharing: &model.Sharing{
				SharingDB: &model.SharingDB{ID: "share1"},
				Files:     []string{"/local/video.mp4"},
				Creator:   adminUser,
			},
			path:        "/",
			want:        "/local/video.mp4",
			expectError: false,
		},
		{
			name: "single file - requested with filename",
			sharing: &model.Sharing{
				SharingDB: &model.SharingDB{ID: "share1"},
				Files:     []string{"/local/video.mp4"},
				Creator:   adminUser,
			},
			path:        "/video.mp4",
			want:        "/local/video.mp4",
			expectError: false,
		},
		{
			name: "single folder - root path",
			sharing: &model.Sharing{
				SharingDB: &model.SharingDB{ID: "share2"},
				Files:     []string{"/local/movies"},
				Creator:   adminUser,
			},
			path:        "/",
			want:        "/local/movies",
			expectError: false,
		},
		{
			name: "single folder - sub path unwrapped",
			sharing: &model.Sharing{
				SharingDB: &model.SharingDB{ID: "share2"},
				Files:     []string{"/local/movies"},
				Creator:   adminUser,
			},
			path:        "/action/movie.mp4",
			want:        "/local/movies/action/movie.mp4",
			expectError: false,
		},
		{
			name: "single folder - requested with folder prefix",
			sharing: &model.Sharing{
				SharingDB: &model.SharingDB{ID: "share2"},
				Files:     []string{"/local/movies"},
				Creator:   adminUser,
			},
			path:        "/movies/action/movie.mp4",
			want:        "/local/movies/action/movie.mp4",
			expectError: false,
		},
		{
			name: "multi files - child match",
			sharing: &model.Sharing{
				SharingDB: &model.SharingDB{ID: "share3"},
				Files:     []string{"/local/a.txt", "/local/b.txt"},
				Creator:   adminUser,
			},
			path:        "/a.txt",
			want:        "/local/a.txt",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := op.GetSharingUnwrapPath(tt.sharing, tt.path)
			if (err != nil) != tt.expectError {
				t.Fatalf("unexpected error state: got %v, want error: %v", err, tt.expectError)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
