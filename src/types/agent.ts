/**
 * Agent types and interfaces for the HashDom frontend.
 * 
 * @packageDocumentation
 */

/**
 * Represents a registered agent in the system.
 * 
 * @interface Agent
 * @property {number} id - Unique identifier for the agent
 * @property {string} name - Display name of the agent
 * @property {'inactive' | 'active' | 'error'} status - Current agent status
 * @property {string} lastHeartbeat - ISO timestamp of last heartbeat
 * @property {number} createdBy - User ID of agent creator
 * @property {string} createdAt - ISO timestamp of creation
 * @property {string} updatedAt - ISO timestamp of last update
 * @property {AgentMetrics} [metrics] - Optional current metrics
 */
export interface Agent {
    id: number;
    name: string;
    status: 'inactive' | 'active' | 'error';
    lastHeartbeat: string;
    createdBy: number;
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