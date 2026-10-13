// Package cosstore wraps authenticated COS operations; credentials never reach logs.
package cosstore

import (
	"context"
	"errors"
	cos "github.com/tencentyun/cos-go-sdk-v5"
	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
	"io"
	"net/http"
	"net/url"
	"peptide-site/internal/conf"
	"strconv"
	"strings"
	"time"
)

type Location struct{ Bucket, Region string }
type Credential struct {
	SecretID    string `json:"secretId"`
	SecretKey   string `json:"secretKey"`
	Token       string `json:"token"`
	StartTime   int    `json:"startTime"`
	ExpiredTime int    `json:"expiredTime"`
}

// Store is consumed by media orchestration. Tests replace the network, never PostgreSQL.
type Store interface {
	Authorize(context.Context, Location, []string, int64, time.Duration) (Credential, error)
	Read(context.Context, Location, string, int64) ([]byte, string, error)
	Promote(context.Context, Location, string, string, string, string) error
	Delete(context.Context, Location, string) error
}
type Client struct{ config conf.COS }

func New(c conf.COS) *Client { return &Client{c} }
func (c *Client) client(l Location) *cos.Client {
	u := &url.URL{Scheme: "https", Host: l.Bucket + ".cos." + l.Region + ".myqcloud.com"}
	return cos.NewClient(&cos.BaseURL{BucketURL: u}, &http.Client{Timeout: 20 * time.Second, Transport: &cos.AuthorizationTransport{SecretID: c.config.SecretID, SecretKey: c.config.SecretKey, Transport: http.DefaultTransport}})
}

type contextTransport struct{ ctx context.Context }

func (t contextTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return http.DefaultTransport.RoundTrip(r.WithContext(t.ctx))
}
func Policy(l Location, keys []string, maxBytes int64) *sts.CredentialPolicy {
	appID := l.Bucket[strings.LastIndex(l.Bucket, "-")+1:]
	resources := make([]string, 0, len(keys))
	for _, key := range keys {
		resources = append(resources, "qcs::cos:"+l.Region+":uid/"+appID+":"+l.Bucket+"/"+key)
	}
	// Images are bounded at 64 MiB, so direct PUT avoids unfinished multipart uploads.
	return &sts.CredentialPolicy{Version: "2.0", Statement: []sts.CredentialPolicyStatement{{Effect: "allow", Action: []string{"name/cos:PutObject"}, Resource: resources, Condition: map[string]map[string]interface{}{"numeric_less_than_equal": {"cos:content-length": maxBytes}}}}}
}
func (c *Client) Authorize(ctx context.Context, l Location, keys []string, maxBytes int64, ttl time.Duration) (Credential, error) {
	cl := sts.NewClient(c.config.SecretID, c.config.SecretKey, &http.Client{Timeout: 10 * time.Second, Transport: contextTransport{ctx}})
	out, err := cl.GetCredential(&sts.CredentialOptions{Policy: Policy(l, keys, maxBytes), Region: l.Region, DurationSeconds: int64(ttl.Seconds())})
	if err != nil || out == nil || out.Credentials == nil {
		return Credential{}, errors.New("COS temporary authorization unavailable")
	}
	return Credential{out.Credentials.TmpSecretID, out.Credentials.TmpSecretKey, out.Credentials.SessionToken, out.StartTime, out.ExpiredTime}, nil
}

var ErrInvalidObject = errors.New("invalid COS object")

func (c *Client) Read(ctx context.Context, l Location, key string, limit int64) ([]byte, string, error) {
	cl := c.client(l)
	head, err := cl.Object.Head(ctx, key, nil)
	if err != nil {
		return nil, "", errors.New("COS object unavailable")
	}
	head.Body.Close()
	size, err := strconv.ParseInt(head.Header.Get("Content-Length"), 10, 64)
	if err != nil || size <= 0 || size > limit {
		return nil, "", ErrInvalidObject
	}
	etag := head.Header.Get("ETag")
	if etag == "" {
		return nil, "", ErrInvalidObject
	}
	headers := http.Header{"If-Match": []string{etag}}
	res, err := cl.Object.Get(ctx, key, &cos.ObjectGetOptions{XOptionHeader: &headers})
	if err != nil {
		return nil, "", errors.New("COS object changed or unavailable")
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil {
		return nil, "", errors.New("COS object read failed")
	}
	if int64(len(body)) != size {
		return nil, "", ErrInvalidObject
	}
	return body, etag, nil
}
func (c *Client) Promote(ctx context.Context, l Location, source, target, etag, mime string) error {
	_, _, err := c.client(l).Object.Copy(ctx, target, l.Bucket+".cos."+l.Region+".myqcloud.com/"+source, &cos.ObjectCopyOptions{ObjectCopyHeaderOptions: &cos.ObjectCopyHeaderOptions{XCosCopySourceIfMatch: etag, XCosMetadataDirective: "Replaced", ContentType: mime, CacheControl: "public, max-age=86400", ContentDisposition: "inline"}})
	if err != nil {
		return errors.New("COS image publication failed")
	}
	return nil
}
func (c *Client) Delete(ctx context.Context, l Location, key string) error {
	_, err := c.client(l).Object.Delete(ctx, key, nil)
	if err != nil {
		var e *cos.ErrorResponse
		if errors.As(err, &e) && e.Response.StatusCode == 404 {
			return nil
		}
		return errors.New("COS cleanup failed")
	}
	return nil
}
