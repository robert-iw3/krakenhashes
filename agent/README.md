# HashDom Agent

The HashDom Agent is responsible for hardware monitoring and management.

## Dependencies

The agent requires certain system dependencies based on your hardware configuration. The agent will automatically detect your hardware and provide installation instructions for missing dependencies.

### Common Dependencies

These are required for basic functionality:

- `pciutils` - PCI device detection
- `util-linux` - CPU information
- `lm-sensors` - Temperature monitoring (optional)

### GPU-Specific Dependencies

The agent will only require dependencies for the GPU vendors present in your system:

#### NVIDIA GPUs
- `nvidia-driver` - NVIDIA driver and tools
- Required for NVIDIA GPU detection and monitoring

#### AMD GPUs
- `rocm-smi` - ROCm tools for AMD GPUs
- `amdgpu-pro` - AMDGPU-PRO driver tools
- At least one of these is required for AMD GPU detection and monitoring

#### Intel GPUs
- `intel-gpu-tools` - Intel GPU tools
- Required for Intel GPU detection and monitoring

### Installation

The agent will check for required dependencies on startup and provide installation instructions if needed. You can also manually check dependencies:

```bash
# Install common dependencies
sudo apt-get install pciutils util-linux lm-sensors

# For NVIDIA GPUs
sudo apt-get install nvidia-driver

# For AMD GPUs (choose one)
sudo apt-get install rocm-smi
# or
sudo apt-get install amdgpu-pro

# For Intel GPUs
sudo apt-get install intel-gpu-tools
```

### Virtualization Support

The agent supports virtualized environments with proper PCI passthrough configuration. Ensure your virtualization platform is configured to expose GPU devices directly to the virtual machine.

## Usage

[Add usage instructions here]

## Development

[Add development instructions here]