/**
 * Agent types and interfaces for the KrakenHashes frontend.
 * 
 * @packageDocumentation
 */

/**
 * Represents a registered agent in the system.
 * 
 * @interface Agent
 * @property {string} id - Unique identifier for the agent (UUID)
 * @property {string} name - Display name of the agent
 * @property {'inactive' | 'active' | 'error'} status - Current agent status
 * @property {string} lastHeartbeat - ISO timestamp of last heartbeat
 * @property {number} createdBy - User ID of agent creator
 * @property {string} createdAt - ISO timestamp of creation
 * @property {string} updatedAt - ISO timestamp of last update
 * @property {AgentMetrics} [metrics] - Optional current metrics
 */
export interface Agent {
    id: string;
    name: string;
    status: 'inactive' | 'active' | 'error';
    lastHeartbeat: string;
    createdBy: {
        id: string;
        username: string;
    };
    version: string;
    hardware: AgentHardware;
    teams: {
        id: string;
        name: string;
    }[];
    createdAt: string;
    updatedAt: string;
    metrics?: AgentMetrics;
}

/**
 * Represents real-time metrics data from an agent.
 * 
 * @interface AgentMetrics
 * @property {number} cpuUsage - CPU usage percentage (0-100)
 * @property {number} gpuUsage - GPU usage percentage (0-100)
 * @property {number} gpuTemp - GPU temperature in Celsius
 * @property {number} memoryUsage - Memory usage percentage (0-100)
 * @property {string} timestamp - ISO timestamp of metrics collection
 */
export interface AgentMetrics {
    cpuUsage: number;
    gpuUsage: number;
    gpuTemp: number;
    memoryUsage: number;
    timestamp: string;
}

/**
 * Form data structure for registering a new agent.
 * 
 * @interface AgentRegistrationForm
 * @property {string} name - Desired name for the new agent
 * @property {number} teamId - Team ID to associate with the agent
 * @property {boolean} continuous - Whether the claim code can be used multiple times
 */
export interface AgentRegistrationForm {
    name: string;
    teamId: number;
    continuous: boolean;
}

/**
 * Structure for agent claim codes.
 * 
 * @interface ClaimCode
 * @property {string} code - The generated claim code
 * @property {boolean} continuous - Whether the code can be used multiple times
 * @property {string} createdAt - ISO timestamp of code generation
 */
export interface ClaimCode {
    code: string;
    continuous: boolean;
    createdAt: string;
}

/**
 * Represents hardware information reported by an agent
 */
export interface AgentHardware {
    cpus: {
        model: string;
        cores: number;
        threads: number;
    }[];
    gpus: {
        model: string;
        memory: string;
        driver: string;
    }[];
    networkInterfaces: {
        name: string;
        ipAddress: string;
    }[];
}

/**
 * Represents a claim voucher in the system
 */
export interface ClaimVoucher {
    code: string;
    created_by: {
        id: string;
        username: string;
        email: string;
        role: string;
    };
    created_by_id: string;
    is_continuous: boolean;
    is_active: boolean;
    created_at: string;
    updated_at: string;
    used_at?: {
        Time: string;
        Valid: boolean;
    };
    used_by_agent_id?: {
        Int64: number;
        Valid: boolean;
    };
} 