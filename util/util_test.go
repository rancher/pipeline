package util

import "testing"

func TestListHooks(t *testing.T) {
	hooks, err := ListWebhook("gitlawr", "php", "ec6c368a6421035b4b5076a5043c89a125dce20f")
	if err != nil {
		t.Logf("get err:%v", err)
	}
	t.Logf("get hooks:%v", hooks)

}

func TestCreateHook(t *testing.T) {
	id, err := CreateWebhook("gitlawr", "php", "ec6c368a6421035b4b5076a5043c89a125dce20f", "testurl", "testsecret")
	t.Logf("get id,err:%v,%v", id, err)
}
