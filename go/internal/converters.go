// Copyright Valkey GLIDE Project Contributors - SPDX Identifier: Apache-2.0

package internal

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/valkey-io/valkey-glide/go/v2/models"
	"github.com/valkey-io/valkey-glide/go/v2/options"
)

func ConvertArrayOfNilOr[T any](data any) (any, error) {
	arr := data.([]any)
	res := make([]models.Result[T], 0, len(arr))

	for _, value := range arr {
		if value == nil {
			res = append(res, models.CreateNilResultOf[T]())
		} else {
			if val, ok := value.(T); ok {
				res = append(res, models.CreateResultOf[T](val))
			} else {
				return nil, fmt.Errorf("unexpected type: %T, expected: %v", val, GetType[T]())
			}
		}
	}
	return any(res), nil
}

func ConvertArrayOf[T any](data any) (any, error) {
	return arrayConverter[T]{
		nil,
		false,
	}.convert(data)
	// actually returns a []T
}

func ConvertMapOf[T any](data any) (any, error) {
	return mapConverter[T]{
		nil,
		false,
	}.convert(data)
	// actually returns a map[string]T
}

// BZPOPMAX BZPOPMIN
func ConvertKeyWithMemberAndScore(data any) (any, error) {
	arr := data.([]any)
	key := arr[0].(string)
	member := arr[1].(string)
	score := arr[2].(float64)
	return models.KeyWithMemberAndScore{Key: key, Member: member, Score: score}, nil
}

// ZMPOP BZMPOP
func ConvertKeyWithArrayOfMembersAndScores(data any) (any, error) {
	if data == nil {
		return nil, nil
	}

	arr := data.([]any)
	key := arr[0].(string)
	memberAndScoreArray, err := MakeConvertMapOfMemberAndScore(false)(arr[1])

	return models.CreateKeyWithArrayOfMembersAndScoresResult(
		models.KeyWithArrayOfMembersAndScores{Key: key, MembersAndScores: memberAndScoreArray.([]models.MemberAndScore)},
	), err
}

// ZRangeWithScores ZInterWithScores ZDiffWithScores ZUnionWithScores
func MakeConvertMapOfMemberAndScore(reverse bool) func(data any) (any, error) {
	return func(data any) (any, error) {
		converted, err := ConvertMapOf[float64](data)
		if err != nil {
			return nil, err
		}

		res := converted.(map[string]float64)
		memberAndScoreArray := make([]models.MemberAndScore, 0, len(res))

		for k, v := range res {
			memberAndScoreArray = append(memberAndScoreArray, models.MemberAndScore{Member: k, Score: v})
		}
		if !reverse {
			sort.Slice(memberAndScoreArray, func(i, j int) bool {
				if memberAndScoreArray[i].Score == memberAndScoreArray[j].Score {
					return memberAndScoreArray[i].Member < memberAndScoreArray[j].Member
				}
				return memberAndScoreArray[i].Score < memberAndScoreArray[j].Score
			})
		} else {
			sort.Slice(memberAndScoreArray, func(i, j int) bool {
				if memberAndScoreArray[i].Score == memberAndScoreArray[j].Score {
					return memberAndScoreArray[i].Member > memberAndScoreArray[j].Member
				}
				return memberAndScoreArray[i].Score > memberAndScoreArray[j].Score
			})
		}

		return memberAndScoreArray, nil
	}
}

// ZRandMemberWithCountWithScores
func ConvertArrayOfMemberAndScore(data any) (any, error) {
	converted, err := arrayConverter[[]any]{
		arrayConverter[any]{
			nil,
			false,
		},
		false,
	}.convert(data)
	if err != nil {
		return nil, err
	}
	pairs := converted.([][]any)
	memberAndScoreArray := make([]models.MemberAndScore, 0, len(pairs))
	for _, pair := range pairs {
		memberAndScoreArray = append(
			memberAndScoreArray,
			models.MemberAndScore{Member: pair[0].(string), Score: pair[1].(float64)},
		)
	}
	return memberAndScoreArray, nil
}

// XAutoClaim XAutoClaimWithOptions
func ConvertXAutoClaimResponse(data any) (any, error) {
	arr := data.([]any)
	len := len(arr)
	if len < 2 || len > 3 {
		return nil, fmt.Errorf("unexpected response array length: %d", len)
	}
	claimedEntries, err := MakeConvertStreamEntryArray(false)(arr[1])
	if err != nil {
		return nil, err
	}
	var deletedMessages []string = nil
	if len == 3 {
		converted, err := arrayConverter[string]{
			nil,
			false,
		}.convert(arr[2])
		if err != nil {
			return nil, err
		}
		deletedMessages = converted.([]string)
	}
	return models.XAutoClaimResponse{
		NextEntry:       arr[0].(string),
		ClaimedEntries:  claimedEntries.([]models.StreamEntry),
		DeletedMessages: deletedMessages,
	}, nil
}

// XAutoClaimJustId XAutoClaimJustIdWithOptions
func ConvertXAutoClaimJustIdResponse(data any) (any, error) {
	arr := data.([]any)
	len := len(arr)
	if len < 2 || len > 3 {
		return nil, fmt.Errorf("unexpected response array length: %d", len)
	}
	converted, err := arrayConverter[string]{
		nil,
		false,
	}.convert(arr[1])
	if err != nil {
		return nil, err
	}

	claimedEntries := converted.([]string)
	var deletedMessages []string = nil
	if len == 3 {
		converted, err = arrayConverter[string]{
			nil,
			false,
		}.convert(arr[2])
		if err != nil {
			return nil, err
		}
		deletedMessages = converted.([]string)
	}
	return models.XAutoClaimJustIdResponse{
		NextEntry:       arr[0].(string),
		ClaimedEntries:  claimedEntries,
		DeletedMessages: deletedMessages,
	}, nil
}

// XInfoConsumers
func ConvertXInfoConsumersResponse(data any) (any, error) {
	converted, err := arrayConverter[map[string]any]{
		nil,
		false,
	}.convert(data)
	if err != nil {
		return nil, err
	}
	arr := converted.([]map[string]any)

	result := make([]models.XInfoConsumerInfo, 0, len(arr))

	for _, group := range arr {
		info := models.XInfoConsumerInfo{
			Name:    group["name"].(string),
			Pending: group["pending"].(int64),
			Idle:    group["idle"].(int64),
		}
		switch inactive := group["inactive"].(type) {
		case int64:
			info.Inactive = models.CreateInt64Result(inactive)
		default:
			info.Inactive = models.CreateNilInt64Result()
		}
		result = append(result, info)
	}

	return result, nil
}

// XInfoGroups
func ConvertXInfoGroupsResponse(data any) (any, error) {
	converted, err := arrayConverter[map[string]any]{
		nil,
		false,
	}.convert(data)
	if err != nil {
		return nil, err
	}
	arr := converted.([]map[string]any)

	result := make([]models.XInfoGroupInfo, 0, len(arr))

	for _, group := range arr {
		info := models.XInfoGroupInfo{
			Name:            group["name"].(string),
			Consumers:       group["consumers"].(int64),
			Pending:         group["pending"].(int64),
			LastDeliveredId: group["last-delivered-id"].(string),
		}
		switch lag := group["lag"].(type) {
		case int64:
			info.Lag = models.CreateInt64Result(lag)
		default:
			info.Lag = models.CreateNilInt64Result()
		}
		switch entriesRead := group["entries-read"].(type) {
		case int64:
			info.EntriesRead = models.CreateInt64Result(entriesRead)
		default:
			info.EntriesRead = models.CreateNilInt64Result()
		}
		result = append(result, info)
	}

	return result, nil
}

// XPending
func ConvertXPendingResponse(data any) (any, error) {
	arr := data.([]any)

	NumOfMessages := arr[0].(int64)
	var StartId, EndId models.Result[string]
	if arr[1] == nil {
		StartId = models.CreateNilStringResult()
	} else {
		StartId = models.CreateStringResult(arr[1].(string))
	}
	if arr[2] == nil {
		EndId = models.CreateNilStringResult()
	} else {
		EndId = models.CreateStringResult(arr[2].(string))
	}

	if pendingMessages, ok := arr[3].([]any); ok {
		var ConsumerPendingMessages []models.ConsumerPendingMessage
		for _, msg := range pendingMessages {
			consumerMessage := msg.([]any)
			count, err := strconv.ParseInt(consumerMessage[1].(string), 10, 64)
			if err != nil {
				return nil, err
			}
			ConsumerPendingMessages = append(ConsumerPendingMessages, models.ConsumerPendingMessage{
				ConsumerName: consumerMessage[0].(string),
				MessageCount: count,
			})
		}
		return models.XPendingSummary{
			NumOfMessages:    NumOfMessages,
			StartId:          StartId,
			EndId:            EndId,
			ConsumerMessages: ConsumerPendingMessages,
		}, nil
	} else {
		return models.XPendingSummary{NumOfMessages: NumOfMessages, StartId: StartId, EndId: EndId, ConsumerMessages: make([]models.ConsumerPendingMessage, 0)}, nil
	}
}

// XPendingWithOptions
func ConvertXPendingWithOptionsResponse(data any) (any, error) {
	arr := data.([]any)
	pendingDetails := make([]models.XPendingDetail, 0, len(arr))

	for _, message := range arr {
		detail := message.([]any)

		pDetail := models.XPendingDetail{
			Id:            detail[0].(string),
			ConsumerName:  detail[1].(string),
			IdleTime:      detail[2].(int64),
			DeliveryCount: detail[3].(int64),
		}
		pendingDetails = append(pendingDetails, pDetail)
	}
	return pendingDetails, nil
}

func Convert2DArrayOfString(data any) (any, error) {
	return arrayConverter[[]string]{
		arrayConverter[string]{
			nil,
			false,
		},
		false,
	}.convert(data)
	// actually returns a [][]string
}

// GeoPos - array of ([]float64 or nil)
func Convert2DArrayOfFloat(data any) (any, error) {
	return arrayConverter[[]float64]{
		arrayConverter[float64]{
			nil,
			true,
		},
		false,
	}.convert(data)
	// actually returns a [][]float64
}

// GeoSearchWithFullOptions
func ConvertLocationArrayResponse(data any) (any, error) {
	converted, err := arrayConverter[[]any]{
		arrayConverter[any]{
			nil,
			false,
		},
		false,
	}.convert(data)
	if err != nil {
		return nil, err
	}

	result := make([]options.Location, 0, len(converted.([][]any)))
	for _, responseArray := range converted.([][]any) {
		location := options.Location{
			Name: responseArray[0].(string),
		}

		additionalData := responseArray[1].([]any)
		for _, value := range additionalData {
			if v, ok := value.(float64); ok {
				location.Dist = v
			}
			if v, ok := value.(int64); ok {
				location.Hash = v
			}
			if coordArray, ok := value.([]any); ok {
				location.Coord = options.GeospatialData{
					Longitude: coordArray[0].(float64),
					Latitude:  coordArray[1].(float64),
				}
			}
		}
		result = append(result, location)
	}

	return result, nil
}

// FunctionList
func ConvertFunctionListResponse(data any) (any, error) {
	result := make([]models.LibraryInfo, 0, len(data.([]any)))
	for _, item := range data.([]any) {
		if itemMap, ok := item.(map[string]any); ok {
			items := itemMap["functions"].([]any)
			functionInfo := make([]models.FunctionInfo, 0, len(items))
			for _, item := range items {
				if function, ok := item.(map[string]any); ok {
					// Handle nullable description
					var description string
					if desc, ok := function["description"].(string); ok {
						description = desc
					}

					// Handle flags map
					flags := make([]string, 0)
					if flagsMap, ok := function["flags"].(map[string]struct{}); ok {
						for flag := range flagsMap {
							flags = append(flags, flag)
						}
					}

					functionInfo = append(functionInfo, models.FunctionInfo{
						Name:        function["name"].(string),
						Description: description,
						Flags:       flags,
					})
				}
			}

			libraryInfo := models.LibraryInfo{
				Name:      itemMap["library_name"].(string),
				Engine:    itemMap["engine"].(string),
				Functions: functionInfo,
			}
			// Handle optional library_code field
			if code, ok := itemMap["library_code"].(string); ok {
				libraryInfo.Code = code
			}
			result = append(result, libraryInfo)
		}
	}
	return result, nil
}

// XRange, XRangeWithOptions, XRevRange, XRevRangeWithOptions
func MakeConvertStreamEntryArray(reverse bool) func(data any) (any, error) {
	return func(data any) (any, error) {
		claimedEntries := data.(map[string]any)
		result := make([]models.StreamEntry, 0, len(claimedEntries))

		for k, v := range claimedEntries {
			result = append(result, models.StreamEntry{ID: k, Fields: CreateFieldInfoArray(v)})
		}
		sort.Slice(result, func(i, j int) bool {
			if reverse {
				return result[i].ID > result[j].ID
			}
			return result[i].ID < result[j].ID
		})
		return result, nil
	}
}

func ConvertFunctionStatsResponse(data any) (any, error) {
	nodeMap := data.(map[string]any)
	// Process engines
	engines := make(map[string]models.Engine)
	if enginesMap, ok := nodeMap["engines"].(map[string]any); ok {
		for engineName, engineData := range enginesMap {
			if engineMap, ok := engineData.(map[string]any); ok {
				engine := models.Engine{
					Language:      engineName,
					FunctionCount: engineMap["functions_count"].(int64),
					LibraryCount:  engineMap["libraries_count"].(int64),
				}
				engines[engineName] = engine
			}
		}
	}

	// Process running script
	var runningScript models.RunningScript
	if scriptData := nodeMap["running_script"]; scriptData != nil {
		if scriptMap, ok := scriptData.(map[string]any); ok {
			runningScript = models.RunningScript{
				Name:     scriptMap["name"].(string),
				Cmd:      scriptMap["command"].(string),
				Args:     scriptMap["arguments"].([]string),
				Duration: time.Duration(scriptMap["duration_ms"].(int64)) * time.Millisecond,
			}
		}
	}

	return models.FunctionStatsResult{
		Engines:       engines,
		RunningScript: runningScript,
	}, nil
}

func ConvertScanResult(data any) (any, error) {
	arr := data.([]any)
	scanData, err := ConvertArrayOf[string](arr[1])
	return models.ScanResult{Cursor: models.NewCursorFromString(arr[0].(string)), Data: scanData.([]string)}, err
}

func ConvertLCSResult(data any) (any, error) {
	lcsResp := data.(map[string]any)
	lenVal, err := ConvertToInt64(lcsResp["len"])
	if err != nil {
		return nil, fmt.Errorf("expected len to be a number, got %T", lcsResp["len"])
	}

	// Parse the matches array using the helper function
	matches, err := ParseLCSMatchedPositions(lcsResp["matches"])
	if err != nil {
		return nil, err
	}
	return models.LCSMatch{
		MatchString: models.DefaultStringResponse,
		Matches:     matches,
		Len:         lenVal,
	}, nil
}

func ConverterAndTypeChecker(
	data any,
	expectedType reflect.Kind,
	isNilable bool,
	converter func(res any) (any, error),
) (any, error) {
	if data == nil {
		if isNilable {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected return type from Glide: got nil, expected %v", expectedType)
	}
	if reflect.TypeOf(data).Kind() == expectedType {
		return converter(data)
	}
	if reflect.TypeOf(data) == reflect.TypeOf(errors.New("")) {
		// not converting a server error
		return data, nil
	}
	// data lost even though it was incorrect
	// TODO maybe still return the data?
	return nil, fmt.Errorf("unexpected return type from Glide: got %v, expected %v", reflect.TypeOf(data), expectedType)
}

// LMPop, LMPopCount, BLMPop, BLMPopCount
func ConvertKeyValuesArrayOrNil(data any) ([]models.KeyValues, error) {
	return keyValuesConverter{canBeNil: true}.convert(data)
}

func ConvertKeyValuesArrayOrNilForBatch(data any) (any, error) {
	return ConvertKeyValuesArrayOrNil(data)
}

// XRead, XReadGroup
func ConvertXReadResponse(data any) (any, error) {
	result := make(map[string]models.StreamResponse)
	// Process the map data directly
	streamMap := data.(map[string]any)
	for streamName, streamData := range streamMap {
		streamResponse := models.StreamResponse{
			Entries: make([]models.StreamEntry, 0),
		}
		// Process fields
		for id, entriesArray := range streamData.(map[string]any) {
			// Process stream entries
			fieldInfos := CreateFieldInfoArray(entriesArray)
			streamResponse.Entries = append(streamResponse.Entries, models.StreamEntry{
				ID:     id,
				Fields: fieldInfos,
			})
		}

		result[streamName] = streamResponse
	}
	return result, nil
}

func ConvertXClaimResponse(data any) (any, error) {
	result := make(map[string]models.XClaimResponse)

	// Process the map data directly
	claimMap, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type received: %T", data)
	}

	for id, entriesArray := range claimMap {
		// Process fields
		fieldInfos := CreateFieldInfoArray(entriesArray)
		result[id] = models.XClaimResponse{
			Fields: fieldInfos,
		}
	}

	return result, nil
}

// ZRankWithScore and ZRevRankWithScore
func ConvertRankAndScoreResponse(data any) (any, error) {
	arr := data.([]any)
	return models.RankAndScore{Rank: arr[0].(int64), Score: arr[1].(float64)}, nil
}

// XInfoStream
func ConvertXInfoStreamResponse(data any) (any, error) {
	infoMap := data.(map[string]any)

	streamInfo := models.XInfoStreamResponse{}
	// Parse integer fields
	ReadValue(infoMap, "length", &streamInfo.Length)
	ReadValue(infoMap, "radix-tree-keys", &streamInfo.RadixTreeKeys)
	ReadValue(infoMap, "radix-tree-nodes", &streamInfo.RadixTreeNodes)
	ReadValue(infoMap, "groups", &streamInfo.Groups)
	ReadResult(infoMap, "entries-added", &streamInfo.EntriesAdded)

	// Parse string fields
	ReadValue(infoMap, "last-generated-id", &streamInfo.LastGeneratedID)
	ReadResult(infoMap, "max-deleted-entry-id", &streamInfo.MaxDeletedEntryID)

	// Get First Entry
	entry := CreateStreamEntry(infoMap, "first-entry")
	if entry.ID != "" {
		streamInfo.FirstEntry = entry
	}

	entry = CreateStreamEntry(infoMap, "last-entry")
	if entry.ID != "" {
		streamInfo.LastEntry = entry
	}

	return streamInfo, nil
}

// XInfoStreamFullWithOptions
func ConvertXInfoStreamFullResponse(data any) (any, error) {
	infoMap := data.(map[string]any)

	streamInfo := models.XInfoStreamFullOptionsResponse{}

	// Parse integer fields
	ReadValue(infoMap, "length", &streamInfo.Length)
	ReadValue(infoMap, "radix-tree-keys", &streamInfo.RadixTreeKeys)
	ReadValue(infoMap, "radix-tree-nodes", &streamInfo.RadixTreeNodes)
	ReadResult(infoMap, "entries-added", &streamInfo.EntriesAdded)

	// Parse string fields
	ReadValue(infoMap, "last-generated-id", &streamInfo.LastGeneratedID)
	ReadResult(infoMap, "max-deleted-entry-id", &streamInfo.MaxDeletedEntryID)

	// Get First Entry
	entry := CreateStreamEntry(infoMap, "first-entry")
	if entry.ID != "" {
		streamInfo.FirstEntry = entry
	}
	// Get Last Entry
	entry = CreateStreamEntry(infoMap, "last-entry")
	if entry.ID != "" {
		streamInfo.LastEntry = entry
	}

	if groups, ok := infoMap["groups"].([]any); ok {
		groupsArr := make([]models.XInfoStreamGroupInfo, 0, len(groups))
		for _, group := range groups {
			groupInfo := models.XInfoStreamGroupInfo{}
			if groupMap, ok := group.(map[string]any); ok {
				if consumers, ok := groupMap["consumers"].([]any); ok {
					consumersArr := make([]models.XInfoStreamConsumerInfo, 0, len(consumers))
					for _, consumerMap := range consumers {
						consumerInfo := models.XInfoStreamConsumerInfo{}
						if consumer, ok := consumerMap.(map[string]any); ok {
							ReadValue(consumer, "name", &consumerInfo.Name)
							ReadValue(consumer, "seen-time", &consumerInfo.SeenTime)
							ReadResult(consumer, "active-time", &consumerInfo.ActiveTime)
							ReadValue(consumer, "pel-count", &consumerInfo.PelCount)

							if pending, ok := consumer["pending"].([]any); ok {
								pendingConsumerArr := make([]models.ConsumerPendingEntry, 0, len(pending))
								for _, entry := range pending {
									if entryArr, ok := entry.([]any); ok {
										pendingConsumerArr = append(
											pendingConsumerArr,
											models.ConsumerPendingEntry{
												Id:             entryArr[0].(string),
												DeliveredTime:  entryArr[1].(int64),
												DeliveredCount: entryArr[2].(int64),
											},
										)
									}
								}
								consumerInfo.Pending = pendingConsumerArr
							}

							consumersArr = append(consumersArr, consumerInfo)
						}
					}
					groupInfo.Consumers = consumersArr
				}
				ReadValue(groupMap, "name", &groupInfo.Name)
				ReadValue(groupMap, "last-delivered-id", &groupInfo.LastDeliveredId)
				ReadValue(groupMap, "pel-count", &groupInfo.PelCount)
				ReadResult(groupMap, "entries-read", &groupInfo.EntriesRead)
				ReadResult(groupMap, "lag", &groupInfo.Lag)

				if pending, ok := groupMap["pending"].([]any); ok {
					pendingArr := make([]models.PendingEntry, 0, len(pending))
					for _, pendingEntry := range pending {
						if pendingEntryArr, ok := pendingEntry.([]any); ok {
							pendingArr = append(pendingArr, models.PendingEntry{
								Id:             pendingEntryArr[0].(string),
								Name:           pendingEntryArr[1].(string),
								DeliveredTime:  pendingEntryArr[2].(int64),
								DeliveredCount: pendingEntryArr[3].(int64),
							})
						}
					}
					groupInfo.Pending = pendingArr
				}
			}
			groupsArr = append(groupsArr, groupInfo)
		}
		streamInfo.Groups = groupsArr
	}
	if val, ok := infoMap["entries"].([]any); ok {
		entriesArr := make([]models.StreamEntry, 0, len(val))
		for _, entry := range val {
			if streamEntry, ok := entry.([]any); ok && len(streamEntry) > 1 {
				entryInfo := models.StreamEntry{}
				entryInfo.ID = streamEntry[0].(string)
				entryInfo.Fields = CreateFieldInfoArray([]any{streamEntry[1]})
				entriesArr = append(entriesArr, entryInfo)
			}
		}
		streamInfo.Entries = entriesArr
	}

	ReadResult(infoMap, "recorded-first-entry-id", &streamInfo.RecordedFirstEntryId)

	return streamInfo, nil
}
