// internal/handlers/distribute.go
package handlers

import (
	"airdrop-system/internal/services"
	"database/sql"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func DistributeAwards(c *gin.Context) {
	airdropIDStr := c.Param("id")

	var req struct {
		WinPercentage float64 `json:"win_percentage" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换ID
	airdropID, err := strconv.ParseUint(airdropIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid airdrop ID"})
		return
	}

	// 调用分发服务
	winners, err := services.DistributeAwards(uint(airdropID), req.WinPercentage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Awards distributed successfully",
		"winners": len(winners),
		"details": winners,
	})
}

// DistributeAirdrop 处理空投分配请求
func DistributeAirdrop(c *gin.Context) {
	// 获取空投ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的空投ID"})
		return
	}

	// 获取数据库连接
	db := c.MustGet("db").(*sql.DB)

	// 验证空投状态
	var status string
	err = db.QueryRow("SELECT status FROM airdrops WHERE id = ?", id).Scan(&status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "空投不存在"})
		return
	}

	if status != "pending_distribution" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "空投状态不是待分配"})
		return
	}

	// 获取空投信息
	var airdrop struct {
		TotalTokens    float64
		WinProbability float64
	}

	err = db.QueryRow("SELECT total_tokens, win_probability FROM airdrops WHERE id = ?", id).Scan(&airdrop.TotalTokens, &airdrop.WinProbability)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "获取空投信息失败"})
		return
	}

	// 获取所有参与者
	rows, err := db.Query(`
		SELECT id, has_tokens
		FROM participants 
		WHERE airdrop_id = ?
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "获取参与者列表失败"})
		return
	}
	defer rows.Close()

	// 解析参与者数据
	type Participant struct {
		ID        int
		HasTokens bool
	}

	var participants []Participant
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.HasTokens); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "解析参与者数据失败"})
			return
		}
		participants = append(participants, p)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "扫描参与者数据失败"})
		return
	}

	// 计算参与者总数
	participantCount := len(participants)
	if participantCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "没有参与者"})
		return
	}

	// 计算中奖概率和奖励金额
	baseWinProbability := airdrop.WinProbability / 100.0

	// 计算中奖人数（不足1人取1人，不是整数取大于这个数的整数）
	winnerCount := int(float64(participantCount) * baseWinProbability)
	if float64(winnerCount) < float64(participantCount)*baseWinProbability {
		winnerCount++ // 向上取整
	}
	if winnerCount == 0 {
		winnerCount = 1 // 至少有一个中奖者
	}
	if winnerCount > participantCount {
		winnerCount = participantCount // 最多所有参与者都中奖
	}

	// 计算每个中奖者的奖励金额（平均分配）
	rewardPerWinner := airdrop.TotalTokens / float64(winnerCount)

	// 根据持币情况计算中奖概率
	type Candidate struct {
		ID     int
		Weight float64 // 权重，用于随机选择
	}

	var candidates []Candidate
	for _, p := range participants {
		weight := 1.0
		if p.HasTokens {
			weight = 3.0 // 持币者权重是普通参与者的3倍
		}
		candidates = append(candidates, Candidate{
			ID:     p.ID,
			Weight: weight,
		})
	}

	// 随机选择中奖者 在 Go 1.20 之前， rand.Seed() 确实是必要的，因为它用于初始化随机数生成器的种子
	rand.Seed(time.Now().UnixNano())

	var winners []int
	remainingCandidates := make([]Candidate, len(candidates))
	copy(remainingCandidates, candidates)

	for i := 0; i < winnerCount && len(remainingCandidates) > 0; i++ {
		// 计算总权重
		var totalWeight float64
		for _, c := range remainingCandidates {
			totalWeight += c.Weight
		}

		// 生成随机数
		random := rand.Float64() * totalWeight

		// 选择中奖者
		var currentWeight float64
		for j, c := range remainingCandidates {
			currentWeight += c.Weight
			if random <= currentWeight {
				winners = append(winners, c.ID)
				// 从候选列表中移除
				remainingCandidates = append(remainingCandidates[:j], remainingCandidates[j+1:]...)
				break
			}
		}
	}

	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "开始事务失败"})
		return
	}

	// 更新中奖者信息
	now := time.Now()
	for _, winnerID := range winners {
		_, err := tx.Exec(`
			UPDATE participants 
			SET is_winner = true, award_amount = ?, awarded_at = ?, distribution_status = 'pending'
			WHERE id = ?
		`, rewardPerWinner, now, winnerID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "更新中奖者信息失败"})
			return
		}
	}

	// 更新空投状态为已分配
	_, err = tx.Exec("UPDATE airdrops SET status = 'distributed' WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "更新空投状态失败"})
		return
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "提交事务失败"})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"success":           true,
		"message":           "空投分配成功",
		"winner_count":      len(winners),
		"reward_per_winner": rewardPerWinner,
	})
}
