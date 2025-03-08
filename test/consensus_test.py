import requests
import time
import subprocess
import sys
from typing import List, Dict
import json
import binascii
import ed25519
from tqdm import tqdm

class ConsensusTest:
    def __init__(self):
        self.nodes = [
            {"port": "8080", "config": "configs/node1.yaml"},
            {"port": "8081", "config": "configs/node2.yaml"},
            {"port": "8082", "config": "configs/node3.yaml"}
        ]
        self.processes: List[subprocess.Popen] = []
        self.base_url = "http://localhost:{}"

    def start_nodes(self):
        """启动所有节点"""
        print("Starting blockchain nodes...")
        for node in self.nodes:
            cmd = ["./twichain", "-config", node["config"]]
            process = subprocess.Popen(cmd)
            self.processes.append(process)
            print(f"Started node on port {node['port']}")

            # 等待节点启动
            time.sleep(3)

    def stop_nodes(self):
        """停止所有节点"""
        print("\nStopping all nodes...")
        for process in self.processes:
            process.terminate()
            process.wait()
        print("All nodes stopped")

    def sign_message(self, private_key: str, content: str) -> str:
            """使用Ed25519私钥签名内容"""
            try:
                # 解码私钥
                priv_key_bytes = binascii.unhexlify(private_key)
                # 创建签名对象
                signer = ed25519.SigningKey(priv_key_bytes)
                # 签名消息
                message_bytes = content.encode('utf-8')
                signature = signer.sign(message_bytes)
                # 返回十六进制编码的签名
                return binascii.hexlify(signature).decode('ascii')
            except Exception as e:
                print(f"Signing error: {e}")
                print(f"Private key length: {len(private_key)}")
                print(f"Private key: {private_key}")
                print(f"Content: {content}")
                raise

    def create_transaction(self, node_port: str, message: str) -> Dict:
        """在指定节点创建交易"""
        url = f"{self.base_url.format(node_port)}/transactions/new"
        data = {
            "sender": "6adb5500f467f004523d0f9e37acbbdaffc033b5f98fcb6c97fb601060b68f90",
            "receiver": "69c5f684026e6bd3e2a8f175a892ca6858cb9936b3c525ce11b981f848a69fc2",
            "message": message,
            "signature": self.sign_message("d24cb18f2225cdf48f17560d8803e5a4285a8c2b17dd94d6b942cb686ba6a92c6adb5500f467f004523d0f9e37acbbdaffc033b5f98fcb6c97fb601060b68f90", message),  # 简化测试，实际应使用有效签名
            "is_like": False,
            "target_post_id": ""
        }
        
        try:
            response = requests.post(url, json=data)
            return response.json()
        except Exception as e:
            print(f"Error creating transaction on port {node_port}: {e}")
            return None

    def get_chain(self, node_port: str) -> Dict:
        """获取指定节点的区块链"""
        url = f"{self.base_url.format(node_port)}/chain"
        try:
            response = requests.get(url)
            return response.json()
        except Exception as e:
            print(f"Error getting chain from port {node_port}: {e}")
            return None

    def verify_consensus(self) -> bool:
        """验证所有节点是否达成共识"""
        chains = []
        for node in self.nodes:
            chain = self.get_chain(node['port'])
            if chain:
                chains.append(chain)

        if not chains:
            return False

        # 检查所有链的长度和内容是否一致
        first_chain = chains[0]
        for chain in chains[1:]:
            if chain['length'] != first_chain['length']:
                return False
            if json.dumps(chain['chain'], sort_keys=True) != json.dumps(first_chain['chain'], sort_keys=True):
                return False

        return True

    def run_test(self):
        """运行完整的测试流程"""
        try:
            # 启动节点
            self.start_nodes()
            
            print("\nTesting consensus mechanism...")
            
            # 在不同节点创建交易
            print("\nCreating transactions on different nodes...")
            total_transactions = 500
            with tqdm(total=total_transactions, desc="Transactions", unit="tx") as pbar:
                for j in range(500):
                    for i, node in enumerate(self.nodes):
                        response = self.create_transaction(
                            node['port'], 
                            f"Test transaction on node {i+1}, number {j+1}"
                        )
                        # print(f"Transaction created on node {i+1}, number {j+1}: {response}")
                        time.sleep(0.1)
                    pbar.update(1)

            print("\nWaiting for mining...")
            time.sleep(20)

            # 验证共识
            print("\nVerifying consensus...")
            if self.verify_consensus():
                print("\033[32mSUCCESS: All nodes have reached consensus!\033[0m")
            else:
                print("\033[31mFAILURE: Nodes have different chains!\033[0m")

            # 打印每个节点的链状态
            print("\nFinal chain state for each node:")
            for node in self.nodes:
                chain = self.get_chain(node['port'])
                print(f"\nNode on port {node['port']}:")
                print(f"Chain length: {chain['length']}")
                # if chain['chain']:
                #     print("Latest block transactions:")
                #     latest_block = chain['chain'][-1]
                #     for tx in latest_block['transactions']:
                #         print(f"- {tx['message']}")

        finally:
            # 清理
            self.stop_nodes()

def main():
    try:
        tester = ConsensusTest()
        tester.run_test()
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
        tester.stop_nodes()
        sys.exit(1)

if __name__ == "__main__":
    main()