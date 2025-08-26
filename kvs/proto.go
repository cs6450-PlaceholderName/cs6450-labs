package kvs

type Put_Batch_Request struct {
	Data map[string]string
}

type Put_Batch_Response struct {
}

type Get_Batch_Request struct {
	Keys []string
}

type Get_Batch_Response struct {
	Keys   []string
	Values []string
}
