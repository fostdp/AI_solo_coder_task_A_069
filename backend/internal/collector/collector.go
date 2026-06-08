package collector

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"dc-cooling-optimizer/internal/db"
)

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

type Collector struct {
	db            *db.DB
	modbusAddr    string
	deviceConfigs []DeviceConfig
	stopCh        chan struct{}
	dataCh        chan *db.DeviceData
	DataChannel   func(*db.DeviceData)
	txID          uint16

	mu             sync.Mutex
	activeConn     *modbusConn
	idleTimeout    time.Duration
	maxConnAge     time.Duration
	reconnectDelay time.Duration
}

func New(database *db.DB, modbusAddr string) *Collector {
	c := &Collector{
		db:             database,
		modbusAddr:     modbusAddr,
		stopCh:         make(chan struct{}),
		dataCh:         make(chan *db.DeviceData, 500),
		idleTimeout:    60 * time.Second,
		maxConnAge:     30 * time.Minute,
		reconnectDelay: 2 * time.Second,
	}

	for i := 0; i < 8; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     i + 1,
			DeviceID:   i + 1,
			DeviceType: "chiller",
			BaseAddr:   0,
			Addr:       0 + i*20,
		})
	}

	for i := 0; i < 12; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     9 + i,
			DeviceID:   9 + i,
			DeviceType: "cooling_tower",
			BaseAddr:   100,
			Addr:       100 + i*20,
		})
	}

	for i := 0; i < 80; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     21 + i,
			DeviceID:   21 + i,
			DeviceType: "precision_ac",
			BaseAddr:   200,
			Addr:       200 + i*20,
		})
	}

	for i := 0; i < 20; i++ {
		c.deviceConfigs = append(c.deviceConfigs, DeviceConfig{
			UnitID:     101 + i,
			DeviceID:   101 + i,
			DeviceType: "cdu",
			BaseAddr:   400,
			Addr:       400 + i*20,
		})
	}

	return c
}

func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(15 * time.Second)
	defer cleanupTicker.Stop()

	c.collectAll(ctx)

	for {
		select {
		case <-ctx.Done():
			c.closeConn()
			return
		case <-c.stopCh:
			c.closeConn()
			return
		case <-ticker.C:
			c.collectAll(ctx)
		case <-cleanupTicker.C:
			c.cleanupIdleConn()
		}
	}
}

func (c *Collector) Stop() {
	close(c.stopCh)
}

func (c *Collector) collectAll(ctx context.Context) {
	for _, cfg := range c.deviceConfigs {
		data, err := c.collectDevice(ctx, cfg)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		select {
		case c.dataCh <- data:
		default:
		}

		if c.DataChannel != nil {
			c.DataChannel(data)
		}
	}
}

func (c *Collector) getConn() (net.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeConn != nil {
		if time.Since(c.activeConn.createdAt) > c.maxConnAge {
			c.activeConn.conn.Close()
			c.activeConn = nil
		} else if time.Since(c.activeConn.lastUsed) > c.idleTimeout {
			c.activeConn.conn.Close()
			c.activeConn = nil
		} else {
			c.activeConn.lastUsed = time.Now()
			return c.activeConn.conn, nil
		}
	}

	conn, err := net.DialTimeout("tcp", c.modbusAddr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	c.activeConn = &modbusConn{
		conn:      conn,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
	}

	return conn, nil
}

func (c *Collector) invalidateConn() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeConn != nil {
		c.activeConn.conn.Close()
		c.activeConn = nil
	}
}

func (c *Collector) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeConn != nil {
		c.activeConn.conn.Close()
		c.activeConn = nil
	}
}

func (c *Collector) cleanupIdleConn() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.activeConn != nil && time.Since(c.activeConn.lastUsed) > c.idleTimeout {
		c.activeConn.conn.Close()
		c.activeConn = nil
	}
}

func (c *Collector) readModbus(unitID int, startAddr int, count int) ([]uint16, error) {
	conn, err := c.getConn()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.txID++
	txID := c.txID
	c.mu.Unlock()

	req := make([]byte, 12)
	binary.BigEndian.PutUint16(req[0:2], txID)
	binary.BigEndian.PutUint16(req[2:4], 0)
	binary.BigEndian.PutUint16(req[4:6], 6)
	req[6] = byte(unitID)
	req[7] = 0x03
	binary.BigEndian.PutUint16(req[8:10], uint16(startAddr))
	binary.BigEndian.PutUint16(req[10:12], uint16(count))

	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(req); err != nil {
		c.invalidateConn()
		return nil, fmt.Errorf("write: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	header := make([]byte, 7)
	if _, err := conn.Read(header); err != nil {
		c.invalidateConn()
		return nil, fmt.Errorf("read header: %w", err)
	}

	respLen := int(binary.BigEndian.Uint16(header[4:6]))
	pdu := make([]byte, respLen-1)
	if _, err := conn.Read(pdu); err != nil {
		c.invalidateConn()
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

	c.mu.Lock()
	if c.activeConn != nil {
		c.activeConn.lastUsed = time.Now()
	}
	c.mu.Unlock()

	return registers, nil
}

func (c *Collector) readModbusWithRetry(unitID int, startAddr int, count int) ([]uint16, error) {
	registers, err := c.readModbus(unitID, startAddr, count)
	if err != nil {
		log.Printf("collector: modbus read failed unit %d addr %d: %v, retrying after reconnect", unitID, startAddr, err)
		time.Sleep(c.reconnectDelay)

		c.invalidateConn()

		registers, err = c.readModbus(unitID, startAddr, count)
		if err != nil {
			return nil, fmt.Errorf("modbus read failed after retry unit %d: %w", unitID, err)
		}
	}
	return registers, nil
}

func (c *Collector) collectDevice(ctx context.Context, cfg DeviceConfig) (*db.DeviceData, error) {
	registers, err := c.readModbusWithRetry(cfg.UnitID, cfg.Addr, 14)
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

	if err := c.db.InsertDeviceData(ctx, data); err != nil {
		return nil, fmt.Errorf("insert device data: %w", err)
	}

	return data, nil
}
