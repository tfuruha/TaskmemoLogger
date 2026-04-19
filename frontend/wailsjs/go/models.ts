export namespace main {
	
	export class LogEntry {
	    timestamp: string;
	    tags: string[];
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.tags = source["tags"];
	        this.text = source["text"];
	    }
	}

}

