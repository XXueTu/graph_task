package trace

// Repository 追踪仓储接口
type Repository interface {
	// SaveSpan 保存跨度
	SaveSpan(span *Span) error
	
	// FindSpan 根据ID查找跨度
	FindSpan(spanID string) (*Span, error)
	
	// FindTraceSpans 查找追踪的所有跨度
	FindTraceSpans(traceID string) ([]*Span, error)
	
	// DeleteSpan 删除跨度
	DeleteSpan(spanID string) error
	
	// DeleteTrace 删除追踪的所有跨度
	DeleteTrace(traceID string) error
}