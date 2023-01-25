// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.11
// source: video_service/proto/video_service.proto

package proto

import (
	proto "cs426.yale.edu/lab1/failure_injection/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type VideoCoefficients struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// a map of feature id to coefficient representing a sparse vector.
	Coeffs map[int32]uint64 `protobuf:"bytes,1,rep,name=coeffs,proto3" json:"coeffs,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (x *VideoCoefficients) Reset() {
	*x = VideoCoefficients{}
	if protoimpl.UnsafeEnabled {
		mi := &file_video_service_proto_video_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VideoCoefficients) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VideoCoefficients) ProtoMessage() {}

func (x *VideoCoefficients) ProtoReflect() protoreflect.Message {
	mi := &file_video_service_proto_video_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VideoCoefficients.ProtoReflect.Descriptor instead.
func (*VideoCoefficients) Descriptor() ([]byte, []int) {
	return file_video_service_proto_video_service_proto_rawDescGZIP(), []int{0}
}

func (x *VideoCoefficients) GetCoeffs() map[int32]uint64 {
	if x != nil {
		return x.Coeffs
	}
	return nil
}

type GetVideoRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VideoIds []uint64 `protobuf:"varint,1,rep,packed,name=video_ids,json=videoIds,proto3" json:"video_ids,omitempty"`
}

func (x *GetVideoRequest) Reset() {
	*x = GetVideoRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_video_service_proto_video_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetVideoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetVideoRequest) ProtoMessage() {}

func (x *GetVideoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_video_service_proto_video_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetVideoRequest.ProtoReflect.Descriptor instead.
func (*GetVideoRequest) Descriptor() ([]byte, []int) {
	return file_video_service_proto_video_service_proto_rawDescGZIP(), []int{1}
}

func (x *GetVideoRequest) GetVideoIds() []uint64 {
	if x != nil {
		return x.VideoIds
	}
	return nil
}

type VideoInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VideoId uint64 `protobuf:"varint,1,opt,name=video_id,json=videoId,proto3" json:"video_id,omitempty"`
	Title   string `protobuf:"bytes,2,opt,name=title,proto3" json:"title,omitempty"`
	Author  string `protobuf:"bytes,3,opt,name=author,proto3" json:"author,omitempty"`
	Url     string `protobuf:"bytes,4,opt,name=url,proto3" json:"url,omitempty"`
	// ranking coefficients for the video
	VideoCoefficients *VideoCoefficients `protobuf:"bytes,5,opt,name=video_coefficients,json=videoCoefficients,proto3" json:"video_coefficients,omitempty"`
}

func (x *VideoInfo) Reset() {
	*x = VideoInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_video_service_proto_video_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VideoInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VideoInfo) ProtoMessage() {}

func (x *VideoInfo) ProtoReflect() protoreflect.Message {
	mi := &file_video_service_proto_video_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VideoInfo.ProtoReflect.Descriptor instead.
func (*VideoInfo) Descriptor() ([]byte, []int) {
	return file_video_service_proto_video_service_proto_rawDescGZIP(), []int{2}
}

func (x *VideoInfo) GetVideoId() uint64 {
	if x != nil {
		return x.VideoId
	}
	return 0
}

func (x *VideoInfo) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

func (x *VideoInfo) GetAuthor() string {
	if x != nil {
		return x.Author
	}
	return ""
}

func (x *VideoInfo) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (x *VideoInfo) GetVideoCoefficients() *VideoCoefficients {
	if x != nil {
		return x.VideoCoefficients
	}
	return nil
}

type GetVideoResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Videos []*VideoInfo `protobuf:"bytes,1,rep,name=videos,proto3" json:"videos,omitempty"`
}

func (x *GetVideoResponse) Reset() {
	*x = GetVideoResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_video_service_proto_video_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetVideoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetVideoResponse) ProtoMessage() {}

func (x *GetVideoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_video_service_proto_video_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetVideoResponse.ProtoReflect.Descriptor instead.
func (*GetVideoResponse) Descriptor() ([]byte, []int) {
	return file_video_service_proto_video_service_proto_rawDescGZIP(), []int{3}
}

func (x *GetVideoResponse) GetVideos() []*VideoInfo {
	if x != nil {
		return x.Videos
	}
	return nil
}

type GetTrendingVideosRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetTrendingVideosRequest) Reset() {
	*x = GetTrendingVideosRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_video_service_proto_video_service_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetTrendingVideosRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTrendingVideosRequest) ProtoMessage() {}

func (x *GetTrendingVideosRequest) ProtoReflect() protoreflect.Message {
	mi := &file_video_service_proto_video_service_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTrendingVideosRequest.ProtoReflect.Descriptor instead.
func (*GetTrendingVideosRequest) Descriptor() ([]byte, []int) {
	return file_video_service_proto_video_service_proto_rawDescGZIP(), []int{4}
}

type GetTrendingVideosResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// video_ids of the trending videos
	Videos []uint64 `protobuf:"varint,1,rep,packed,name=videos,proto3" json:"videos,omitempty"`
	// unix timestamp of when to consider this trending videos response obsolete
	ExpirationTimeS uint64 `protobuf:"varint,2,opt,name=expiration_time_s,json=expirationTimeS,proto3" json:"expiration_time_s,omitempty"`
}

func (x *GetTrendingVideosResponse) Reset() {
	*x = GetTrendingVideosResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_video_service_proto_video_service_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetTrendingVideosResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTrendingVideosResponse) ProtoMessage() {}

func (x *GetTrendingVideosResponse) ProtoReflect() protoreflect.Message {
	mi := &file_video_service_proto_video_service_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTrendingVideosResponse.ProtoReflect.Descriptor instead.
func (*GetTrendingVideosResponse) Descriptor() ([]byte, []int) {
	return file_video_service_proto_video_service_proto_rawDescGZIP(), []int{5}
}

func (x *GetTrendingVideosResponse) GetVideos() []uint64 {
	if x != nil {
		return x.Videos
	}
	return nil
}

func (x *GetTrendingVideosResponse) GetExpirationTimeS() uint64 {
	if x != nil {
		return x.ExpirationTimeS
	}
	return 0
}

var File_video_service_proto_video_service_proto protoreflect.FileDescriptor

var file_video_service_proto_video_service_proto_rawDesc = []byte{
	0x0a, 0x27, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0d, 0x76, 0x69, 0x64, 0x65, 0x6f,
	0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x2f, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72,
	0x65, 0x5f, 0x69, 0x6e, 0x6a, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72, 0x65, 0x5f, 0x69, 0x6e, 0x6a, 0x65, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x94, 0x01, 0x0a, 0x11, 0x56, 0x69,
	0x64, 0x65, 0x6f, 0x43, 0x6f, 0x65, 0x66, 0x66, 0x69, 0x63, 0x69, 0x65, 0x6e, 0x74, 0x73, 0x12,
	0x44, 0x0a, 0x06, 0x63, 0x6f, 0x65, 0x66, 0x66, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x2c, 0x2e, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x56, 0x69, 0x64, 0x65, 0x6f, 0x43, 0x6f, 0x65, 0x66, 0x66, 0x69, 0x63, 0x69, 0x65, 0x6e, 0x74,
	0x73, 0x2e, 0x43, 0x6f, 0x65, 0x66, 0x66, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x63,
	0x6f, 0x65, 0x66, 0x66, 0x73, 0x1a, 0x39, 0x0a, 0x0b, 0x43, 0x6f, 0x65, 0x66, 0x66, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x22, 0x2e, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x69, 0x64, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x04, 0x52, 0x08, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x49, 0x64, 0x73,
	0x22, 0xb7, 0x01, 0x0a, 0x09, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x19,
	0x0a, 0x08, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x07, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x69, 0x74,
	0x6c, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x6c, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x12, 0x4f, 0x0a, 0x12, 0x76, 0x69, 0x64,
	0x65, 0x6f, 0x5f, 0x63, 0x6f, 0x65, 0x66, 0x66, 0x69, 0x63, 0x69, 0x65, 0x6e, 0x74, 0x73, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x43, 0x6f, 0x65, 0x66, 0x66,
	0x69, 0x63, 0x69, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x11, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x43, 0x6f,
	0x65, 0x66, 0x66, 0x69, 0x63, 0x69, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x44, 0x0a, 0x10, 0x47, 0x65,
	0x74, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x30,
	0x0a, 0x06, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x18,
	0x2e, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x56,
	0x69, 0x64, 0x65, 0x6f, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x06, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x73,
	0x22, 0x1a, 0x0a, 0x18, 0x47, 0x65, 0x74, 0x54, 0x72, 0x65, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x56,
	0x69, 0x64, 0x65, 0x6f, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x5f, 0x0a, 0x19,
	0x47, 0x65, 0x74, 0x54, 0x72, 0x65, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x56, 0x69, 0x64, 0x65, 0x6f,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x76, 0x69, 0x64,
	0x65, 0x6f, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x04, 0x52, 0x06, 0x76, 0x69, 0x64, 0x65, 0x6f,
	0x73, 0x12, 0x2a, 0x0a, 0x11, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f,
	0x74, 0x69, 0x6d, 0x65, 0x5f, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0f, 0x65, 0x78,
	0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65, 0x53, 0x32, 0xb8, 0x02,
	0x0a, 0x0c, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x4b,
	0x0a, 0x08, 0x47, 0x65, 0x74, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x12, 0x1e, 0x2e, 0x76, 0x69, 0x64,
	0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x56, 0x69,
	0x64, 0x65, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1f, 0x2e, 0x76, 0x69, 0x64,
	0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x56, 0x69,
	0x64, 0x65, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x66, 0x0a, 0x11, 0x47,
	0x65, 0x74, 0x54, 0x72, 0x65, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x73,
	0x12, 0x27, 0x2e, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x2e, 0x47, 0x65, 0x74, 0x54, 0x72, 0x65, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x56, 0x69, 0x64, 0x65,
	0x6f, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x28, 0x2e, 0x76, 0x69, 0x64, 0x65,
	0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x54, 0x72, 0x65,
	0x6e, 0x64, 0x69, 0x6e, 0x67, 0x56, 0x69, 0x64, 0x65, 0x6f, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x73, 0x0a, 0x12, 0x53, 0x65, 0x74, 0x49, 0x6e, 0x6a, 0x65, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x2c, 0x2e, 0x66, 0x61, 0x69, 0x6c,
	0x75, 0x72, 0x65, 0x5f, 0x69, 0x6e, 0x6a, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x53, 0x65,
	0x74, 0x49, 0x6e, 0x6a, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2d, 0x2e, 0x66, 0x61, 0x69, 0x6c, 0x75, 0x72,
	0x65, 0x5f, 0x69, 0x6e, 0x6a, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x53, 0x65, 0x74, 0x49,
	0x6e, 0x6a, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x29, 0x5a, 0x27, 0x63, 0x73, 0x34, 0x32,
	0x36, 0x2e, 0x79, 0x61, 0x6c, 0x65, 0x2e, 0x65, 0x64, 0x75, 0x2f, 0x6c, 0x61, 0x62, 0x31, 0x2f,
	0x76, 0x69, 0x64, 0x65, 0x6f, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_video_service_proto_video_service_proto_rawDescOnce sync.Once
	file_video_service_proto_video_service_proto_rawDescData = file_video_service_proto_video_service_proto_rawDesc
)

func file_video_service_proto_video_service_proto_rawDescGZIP() []byte {
	file_video_service_proto_video_service_proto_rawDescOnce.Do(func() {
		file_video_service_proto_video_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_video_service_proto_video_service_proto_rawDescData)
	})
	return file_video_service_proto_video_service_proto_rawDescData
}

var file_video_service_proto_video_service_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_video_service_proto_video_service_proto_goTypes = []interface{}{
	(*VideoCoefficients)(nil),                // 0: video_service.VideoCoefficients
	(*GetVideoRequest)(nil),                  // 1: video_service.GetVideoRequest
	(*VideoInfo)(nil),                        // 2: video_service.VideoInfo
	(*GetVideoResponse)(nil),                 // 3: video_service.GetVideoResponse
	(*GetTrendingVideosRequest)(nil),         // 4: video_service.GetTrendingVideosRequest
	(*GetTrendingVideosResponse)(nil),        // 5: video_service.GetTrendingVideosResponse
	nil,                                      // 6: video_service.VideoCoefficients.CoeffsEntry
	(*proto.SetInjectionConfigRequest)(nil),  // 7: failure_injection.SetInjectionConfigRequest
	(*proto.SetInjectionConfigResponse)(nil), // 8: failure_injection.SetInjectionConfigResponse
}
var file_video_service_proto_video_service_proto_depIdxs = []int32{
	6, // 0: video_service.VideoCoefficients.coeffs:type_name -> video_service.VideoCoefficients.CoeffsEntry
	0, // 1: video_service.VideoInfo.video_coefficients:type_name -> video_service.VideoCoefficients
	2, // 2: video_service.GetVideoResponse.videos:type_name -> video_service.VideoInfo
	1, // 3: video_service.VideoService.GetVideo:input_type -> video_service.GetVideoRequest
	4, // 4: video_service.VideoService.GetTrendingVideos:input_type -> video_service.GetTrendingVideosRequest
	7, // 5: video_service.VideoService.SetInjectionConfig:input_type -> failure_injection.SetInjectionConfigRequest
	3, // 6: video_service.VideoService.GetVideo:output_type -> video_service.GetVideoResponse
	5, // 7: video_service.VideoService.GetTrendingVideos:output_type -> video_service.GetTrendingVideosResponse
	8, // 8: video_service.VideoService.SetInjectionConfig:output_type -> failure_injection.SetInjectionConfigResponse
	6, // [6:9] is the sub-list for method output_type
	3, // [3:6] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_video_service_proto_video_service_proto_init() }
func file_video_service_proto_video_service_proto_init() {
	if File_video_service_proto_video_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_video_service_proto_video_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VideoCoefficients); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_video_service_proto_video_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetVideoRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_video_service_proto_video_service_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VideoInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_video_service_proto_video_service_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetVideoResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_video_service_proto_video_service_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetTrendingVideosRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_video_service_proto_video_service_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetTrendingVideosResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_video_service_proto_video_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_video_service_proto_video_service_proto_goTypes,
		DependencyIndexes: file_video_service_proto_video_service_proto_depIdxs,
		MessageInfos:      file_video_service_proto_video_service_proto_msgTypes,
	}.Build()
	File_video_service_proto_video_service_proto = out.File
	file_video_service_proto_video_service_proto_rawDesc = nil
	file_video_service_proto_video_service_proto_goTypes = nil
	file_video_service_proto_video_service_proto_depIdxs = nil
}
