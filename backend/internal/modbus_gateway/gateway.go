package modbus_gateway

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"dc-cooling-optimizer/internal/config"
	"dc-cooling-optimizer/internal/db"
)

type DeviceDataEvent struct {
	Data        *db.DeviceData
	DeviceType  string
	CollectedAt time.Time
}

type DeviceConfig struct {
	UnitID     int
	DeviceID   int
	DeviceType string
	BaseAddr   int
	Addr       int
}

type modbusConn struct {
	conn      net.Conn
	createdAt time.Time
	lastUsed  time.Time
}

type Gateway struct {
	db            *db.DB
	cfg           *config.ModbusGatewayConfig
	deviceConfigs []DeviceConfig
	outCh         chan *DeviceDataEvent
	stopCh        chan struct{}
	mu            sync.Mutex
	activeConn    *modbusConn
	txID          uint16
}

func New(database *db.DB, cfg *config.ModbusGatewayConfig, deviceConfigs []DeviceConfig) *Gateway {
	buf := cfg.DataChannelBuffer
	if buf <= 0 {
		buf = 500
	}

	g := &Gateway{
		db:            database,
		cfg:           cfg,
		deviceConfigs: deviceConfigs,
		outCh:         make(chan *DeviceDataEvent, buf),
		stopCh:        make(chan struct{}),
	}

	if g.deviceConfigs == nil {
		for i := 0; i < 8; i++ {
			g.deviceConfigs = append(g.deviceConfigs, DeviceConfig{
				UnitID:     i + 1,
				DeviceID:   i + 1,
				DeviceType: "chiller",
				BaseAddr:   0,
				Addr:       0 + i*20,
			})
		}
		for i := 0; i < 12; i++ {
			g.deviceConfigs = append(g.deviceConfigs, DeviceConfig{
				UnitID:     9 + i,
				DeviceID:   9 + i,
				DeviceType: "cooling_tower",
				BaseAddr:   100,
				Addr:       100 + i*20,
			})
		}
		for i := 0; i < 80; i++ {
			g.deviceConfigs = append(g.deviceConfigs, DeviceConfig{
				UnitID:     21 + i,
				DeviceID:   21 + i,
				DeviceType: "precision_ac",
				BaseAddr:   200,
				Addr:       200 + i*20,
			})
		}
		for i := 0; i < 20; i++ {
			g.deviceConfigs = append(g.deviceConfigs, DeviceConfig{
				UnitID:     101 + i,
				DeviceID:   101 + i,
				DeviceType: "cdu",
				BaseAddr:   400,
				Addr:       400 + i*20,
			})
		}
	}

	return g
}

func (g *Gateway) Start(ctx context.Context) {
	collectTicker := time.NewTicker(g.cfg.CollectInterval())
	defer collectTicker.Stop()

	cleanupInterval := g.cfg.IdleTimeout() / 2
	if cleanupInterval < 15*time.Second {
		cleanupInterval = 15 * time.Second
	}
	cleanupTicker := time.NewTicker(cleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.closeConn()
			return
		case <-g.stopCh:
			g.closeConn()
			return
		case <-collectTicker.C:
			g.collectAll(ctx)
		case <-cleanupTicker.C:
			g.cleanupIdleConn()
		}
	}
}

func (g *Gateway) Stop() {
	close(g.stopCh)
}

func (g *Gateway) Output() <-chan *DeviceDataEvent {
	return g.outCh
}

func (g *Gateway) collectAll(ctx context.Context) {
	for _, dc := range g.deviceConfigs {
		data, err := g.collectDevice(ctx, dc)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		select {
		case g.outCh <- &DeviceDataEvent{Data: data, DeviceType: cfg.DeviceType, CollectedAt: time.Now()}:
		default:
		}
	}
}

func (g *Gateway) getConn() (net.Conn, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.activeConn != nil {
		if time.Since(g.activeConn.createdAt) > g.cfg.MaxConnAge() {
			g.activeConn.conn.Close()
			g.activeConn = nil
		} else if time.Since(g.activeConn.lastUsed) > g.cfg.IdleTimeout() {
			g.activeConn.conn.Close()
			g.activeConn = nil
		} else {
			g.activeConn.lastUsed = time.Now()
			return g.activeConn.conn, nil
		}
	}

	conn, err := net.DialTimeout("tcp", g.cfg.Address, g.cfg.DialTimeout())
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	conn.SetDeadline(time.Now().Add(g.cfg.ReadTimeout() + g.cfg.WriteTimeout()))

	g.activeConn = &modbusConn{
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}

	return conn, nil
}

func (g *Gateway) invalidateConn() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.activeConn != nil {
		g.activeConn.conn.Close()
		g.activeConn = nil
	}
}

func (g *Gateway) closeConn() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.activeConn != nil {
		g.activeConn.conn.Close()
		g.activeConn = nil
	}
}

func (g *Gateway) cleanupIdleConn() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.activeConn != nil && time.Since(g.activeConn.lastUsed) > g.cfg.IdleTimeout() {
		g.activeConn.conn.Close()
		g.activeConn = nil
	}
}

func (g *Gateway) readModbus(unitID int, startAddr int, count int) ([]uint16, error) {
	conn, err := g.getConn()
	if err != nil {
		return nil, err
	}

	g.mu.Lock()
	g.txID++
	txID := g.txID
	g.mu.Unlock()

	req := make([]byte, 12)
	binary.BigEndian.PutUint16(req[0:2], txID)
	binary.BigEndian.PutUint16(req[2:4], 0)
	binary.BigEndian.PutUint16(req[4:6], 6)
	req[6] = byte(unitID)
	req[7] = 0x03
	binary.BigEndian.PutUint16(req[8:10], uint16(startAddr))
	binary.BigEndian.PutUint16(req[10:12], uint16(count))

	conn.SetWriteDeadline(time.Now().Add(g.cfg.WriteTimeout()))
	if _, err := conn.Write(req); err != nil {
		g.invalidateConn()
		return nil, fmt.Errorf("write: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(g.cfg.ReadTimeout()))

	header := make([]byte, 7)
	if _, err := conn.Read(header); err != nil {
		g.invalidateConn()
		return nil, fmt.Errorf("read header: %w", err)
	}

	respLen := int(binary.BigEndian.Uint16(header[4:6]))
	pdu := make([]byte, respLen-1)
	if _, err := conn.Read(pdu); err != nil {
		g.invalidateConn()
		return nil, fmt.Errorf("read pdu: %w", err)
	}

	if pdu[0] != byte(unitID) {
		return nil, fmt.Errorf("unit id mismatch: got %d, want %d", pdu[0], unitID)
	}

	if pdu[1] != 0x03 {
		if pdu[1]&0x80 != 0 {
			return nil, fmt.Errorf("modbus exception: code %d", pdu[2])
		}
		return nil, fmt.Errorf("function code mismatch: got %d", pdu[1])
	}

	byteCount := int(pdu[2])
	if byteCount != count*2 {
		return nil, fmt.Errorf("byte count mismatch: got %d, want %d", byteCount, count*2)
	}

	registers := make([]uint16, count)
	for i := 0; i < count; i++ {
		registers[i] = binary.BigEndian.Uint16(pdu[3+i*2 : 5+i*2])
	}

	g.mu.Lock()
	if g.activeConn != nil {
		g.activeConn.lastUsed = time.Now()
	}
	g.mu.Unlock()

	return registers, nil
}

func (g *Gateway) readModbusWithRetry(unitID int, startAddr int, count int) ([]uint16, error) {
	var lastErr error
	for i := 0; i <= g.cfg.RetryCount; i++ {
		registers, err := g.readModbus(unitID, startAddr, count)
		if err == nil {
			return registers, nil
		}
		lastErr = err
		log.Printf("gateway: modbus read failed unit %d addr %d (attempt %d/%d): %v", unitID, startAddr, i+1, g.cfg.RetryCount+1, err)
		if i < g.cfg.RetryCount {
			g.invalidateConn()
			time.Sleep(g.cfg.ReconnectDelay())
		}
	}
	return nil, fmt.Errorf("modbus read failed after %d attempts unit %d: %w", g.cfg.RetryCount+1, unitID, lastErr)
}

func (g *Gateway) collectDevice(ctx context.Context, cfg DeviceConfig) (*db.DeviceData, error) {
	registers, err := g.readModbusWithRetry(cfg.UnitID, cfg.Addr, 14)
	if err != nil {
		return nil, fmt.Errorf("read modbus unit %d: %w", cfg.UnitID, err)
	}

	data := &db.DeviceData{
		Time:            time.Now(),
		DeviceID:        cfg.DeviceID,
		SupplyTemp:      float64(registers[0]) / 10.0,
		ReturnTemp:      float64(registers[2]) / 10.0,
		FlowRate:        float64(registers[4]) / 10.0,
		Power:           float64(registers[6]) / 10.0,
		Pressure:        float64(registers[8]) / 10.0,
		COP:             float64(registers[10]) / 10.0,
		CoolingCapacity: float64(registers[12]) / 10.0,
		Status:          1,
	}

	if err := g.db.InsertDeviceData(ctx, data); err != nil {
		return nil, fmt.Errorf("insert device data: %w", err)
	}

	return data, nil
}
